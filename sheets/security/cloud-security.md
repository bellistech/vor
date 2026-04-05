# Cloud Security

Securing cloud infrastructure, workloads, and data across IaaS, PaaS, and SaaS using IAM, encryption, network controls, posture management, and compliance frameworks.

## Concepts

### Shared Responsibility Model

```
+------------------------------------------------------------------+
|                    Responsibility Matrix                          |
+------------------+----------+----------+----------+--------------+
|                  |  On-Prem |  IaaS    |  PaaS    |  SaaS        |
+------------------+----------+----------+----------+--------------+
| Data             | Customer | Customer | Customer | Customer     |
| Identity/Access  | Customer | Customer | Customer | Customer     |
| Applications     | Customer | Customer | Customer | Provider     |
| Network Controls | Customer | Customer | Shared   | Provider     |
| OS               | Customer | Customer | Provider | Provider     |
| Hypervisor       | Customer | Provider | Provider | Provider     |
| Physical         | Customer | Provider | Provider | Provider     |
+------------------+----------+----------+----------+--------------+
```

- Customer always responsible for: data classification, access policies, compliance
- Provider always responsible for: physical security, hypervisor, global network
- The boundary shifts as you move from IaaS to SaaS

### Cloud Security Posture Management (CSPM)

- Continuously monitors cloud configurations against security best practices
- Detects misconfigurations: public S3 buckets, open security groups, unencrypted storage
- Maps findings to compliance frameworks (CIS, NIST, PCI-DSS, SOC 2)
- Tools: AWS Security Hub, Azure Defender for Cloud, GCP Security Command Center, Prisma Cloud, Wiz

### Cloud Workload Protection Platform (CWPP)

- Protects workloads running in cloud: VMs, containers, serverless functions
- Runtime threat detection, vulnerability scanning, file integrity monitoring
- Agent-based or agentless (API/snapshot-based scanning)
- Covers the compute layer that CSPM does not

### Cloud Access Security Broker (CASB)

```
# CASB deployment modes:
# 1. Forward proxy: inline between user and cloud (requires agent or PAC file)
# 2. Reverse proxy: inline between cloud and user (agentless, SSO integration)
# 3. API-based: out-of-band, connects directly to SaaS APIs (no inline component)

# CASB capabilities (four pillars):
# - Visibility: discover shadow IT, catalog cloud app usage
# - Compliance: DLP, data residency, regulatory controls
# - Data Security: encryption, tokenization, access control
# - Threat Protection: malware detection, anomalous behavior, compromised accounts
```

### Cloud-Native Application Protection Platform (CNAPP)

- Converged platform combining CSPM + CWPP + CIEM + pipeline security
- Shift-left: scan IaC templates and container images in CI/CD pipeline
- Shift-right: runtime protection and continuous posture monitoring
- Single pane of glass across build-time, deploy-time, and runtime

## Cloud IAM

### AWS IAM

```json
// IAM policy structure — JSON policy document
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowS3ReadOnly",
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::my-bucket",
        "arn:aws:s3:::my-bucket/*"
      ],
      "Condition": {
        "IpAddress": {
          "aws:SourceIp": "10.0.0.0/8"
        }
      }
    }
  ]
}
```

```bash
# AWS CLI — IAM operations

# List all IAM users
aws iam list-users

# Create a user with programmatic access
aws iam create-user --user-name deploy-bot
aws iam create-access-key --user-name deploy-bot

# Attach a managed policy to a user
aws iam attach-user-policy \
  --user-name deploy-bot \
  --policy-arn arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess

# Create a role for EC2 instances
aws iam create-role \
  --role-name ec2-s3-access \
  --assume-role-policy-document file://trust-policy.json

# List attached policies for a role
aws iam list-attached-role-policies --role-name ec2-s3-access

# Enable MFA for a user
aws iam enable-mfa-device \
  --user-name admin \
  --serial-number arn:aws:iam::123456789012:mfa/admin \
  --authentication-code1 123456 \
  --authentication-code2 789012

# Simulate policy evaluation (check if action is allowed)
aws iam simulate-principal-policy \
  --policy-source-arn arn:aws:iam::123456789012:user/deploy-bot \
  --action-names s3:GetObject \
  --resource-arns arn:aws:s3:::my-bucket/data.csv
```

### Azure AD (Entra ID)

```bash
# Azure CLI — identity and access

# List all users
az ad user list --output table

# Create a service principal for automation
az ad sp create-for-rbac \
  --name deploy-sp \
  --role Contributor \
  --scopes /subscriptions/<sub-id>/resourceGroups/prod-rg

# Assign a role to a user at resource group scope
az role assignment create \
  --assignee user@company.com \
  --role "Reader" \
  --resource-group prod-rg

# List role assignments for a resource group
az role assignment list \
  --resource-group prod-rg \
  --output table

# Enable Conditional Access (requires Azure AD Premium P1+)
# Typically configured in Azure Portal or via Graph API
# Key policies: require MFA, block legacy auth, device compliance
```

### GCP IAM

```bash
# gcloud — IAM operations

# List IAM policy bindings for a project
gcloud projects get-iam-policy my-project --format=json

# Grant a role to a user
gcloud projects add-iam-policy-binding my-project \
  --member="user:admin@company.com" \
  --role="roles/storage.objectViewer"

# Grant a role to a service account
gcloud projects add-iam-policy-binding my-project \
  --member="serviceAccount:deploy@my-project.iam.gserviceaccount.com" \
  --role="roles/container.developer"

# Create a custom role
gcloud iam roles create customStorageReader \
  --project=my-project \
  --title="Custom Storage Reader" \
  --permissions=storage.objects.get,storage.objects.list

# List service accounts
gcloud iam service-accounts list

# Create a service account key (use Workload Identity Federation instead when possible)
gcloud iam service-accounts keys create key.json \
  --iam-account=deploy@my-project.iam.gserviceaccount.com
```

## Cloud Network Security

### VPC Security Groups (AWS)

```bash
# Security groups — stateful firewall at instance level

# Create a security group
aws ec2 create-security-group \
  --group-name web-sg \
  --description "Web server security group" \
  --vpc-id vpc-abc123

# Allow inbound HTTPS from anywhere
aws ec2 authorize-security-group-ingress \
  --group-id sg-12345 \
  --protocol tcp \
  --port 443 \
  --cidr 0.0.0.0/0

# Allow inbound SSH from specific CIDR only
aws ec2 authorize-security-group-ingress \
  --group-id sg-12345 \
  --protocol tcp \
  --port 22 \
  --cidr 10.0.0.0/8

# Allow all traffic from another security group (inter-tier)
aws ec2 authorize-security-group-ingress \
  --group-id sg-db-tier \
  --protocol tcp \
  --port 5432 \
  --source-group sg-app-tier

# List security group rules
aws ec2 describe-security-group-rules \
  --filters Name=group-id,Values=sg-12345
```

### Network ACLs (AWS)

```bash
# NACLs — stateless firewall at subnet level
# Evaluated in order by rule number (lowest first)

# Allow inbound HTTPS (rule 100)
aws ec2 create-network-acl-entry \
  --network-acl-id acl-abc123 \
  --rule-number 100 \
  --protocol tcp \
  --port-range From=443,To=443 \
  --cidr-block 0.0.0.0/0 \
  --rule-action allow \
  --ingress

# Deny all other inbound (explicit deny, rule 200)
aws ec2 create-network-acl-entry \
  --network-acl-id acl-abc123 \
  --rule-number 200 \
  --protocol -1 \
  --cidr-block 0.0.0.0/0 \
  --rule-action deny \
  --ingress

# Allow outbound ephemeral ports (for return traffic — NACLs are stateless)
aws ec2 create-network-acl-entry \
  --network-acl-id acl-abc123 \
  --rule-number 100 \
  --protocol tcp \
  --port-range From=1024,To=65535 \
  --cidr-block 0.0.0.0/0 \
  --rule-action allow \
  --egress
```

### Service Endpoints and Private Link

```bash
# AWS VPC Endpoints — access AWS services without internet gateway

# Gateway endpoint (S3, DynamoDB — free, route table-based)
aws ec2 create-vpc-endpoint \
  --vpc-id vpc-abc123 \
  --service-name com.amazonaws.us-east-1.s3 \
  --route-table-ids rtb-abc123

# Interface endpoint (most services — uses PrivateLink, ENI-based)
aws ec2 create-vpc-endpoint \
  --vpc-id vpc-abc123 \
  --vpc-endpoint-type Interface \
  --service-name com.amazonaws.us-east-1.secretsmanager \
  --subnet-ids subnet-abc123 \
  --security-group-ids sg-abc123

# Azure Private Endpoint
az network private-endpoint create \
  --name pe-storage \
  --resource-group prod-rg \
  --vnet-name prod-vnet \
  --subnet private-endpoints \
  --private-connection-resource-id /subscriptions/.../storageAccounts/mystorage \
  --group-id blob \
  --connection-name pe-storage-conn

# GCP Private Google Access (for instances without external IPs)
gcloud compute networks subnets update private-subnet \
  --region=us-central1 \
  --enable-private-ip-google-access
```

## Cloud Encryption

### Key Management

```bash
# AWS KMS — create and manage encryption keys

# Create a symmetric encryption key
aws kms create-key \
  --description "Production data encryption key" \
  --key-usage ENCRYPT_DECRYPT \
  --origin AWS_KMS

# Create an alias for easy reference
aws kms create-alias \
  --alias-name alias/prod-data-key \
  --target-key-id <key-id>

# Encrypt data with KMS key
aws kms encrypt \
  --key-id alias/prod-data-key \
  --plaintext fileb://secret.txt \
  --output text --query CiphertextBlob | base64 --decode > encrypted.bin

# Decrypt data
aws kms decrypt \
  --ciphertext-blob fileb://encrypted.bin \
  --output text --query Plaintext | base64 --decode > decrypted.txt

# Envelope encryption — generate a data key
aws kms generate-data-key \
  --key-id alias/prod-data-key \
  --key-spec AES_256
# Returns: Plaintext (use to encrypt data, then discard)
#          CiphertextBlob (encrypted copy of data key, store alongside data)

# Enable automatic key rotation (annual)
aws kms enable-key-rotation --key-id <key-id>
```

```bash
# Azure Key Vault
az keyvault create \
  --name prod-vault \
  --resource-group prod-rg \
  --location eastus \
  --enable-purge-protection true

# Store a secret
az keyvault secret set \
  --vault-name prod-vault \
  --name db-password \
  --value "s3cure-pa55w0rd"

# Create an encryption key
az keyvault key create \
  --vault-name prod-vault \
  --name data-enc-key \
  --kty RSA \
  --size 2048

# GCP Cloud KMS
gcloud kms keyrings create prod-ring --location=global
gcloud kms keys create data-key \
  --location=global \
  --keyring=prod-ring \
  --purpose=encryption
```

## Cloud Logging and Monitoring

### AWS CloudTrail

```bash
# CloudTrail — API audit logging for AWS accounts

# Create a trail logging all management events
aws cloudtrail create-trail \
  --name org-trail \
  --s3-bucket-name cloudtrail-logs-bucket \
  --is-multi-region-trail \
  --enable-log-file-validation

# Start logging
aws cloudtrail start-logging --name org-trail

# Enable data events for S3 (object-level logging)
aws cloudtrail put-event-selectors \
  --trail-name org-trail \
  --event-selectors '[{
    "ReadWriteType": "All",
    "IncludeManagementEvents": true,
    "DataResources": [{
      "Type": "AWS::S3::Object",
      "Values": ["arn:aws:s3:::sensitive-bucket/"]
    }]
  }]'

# Query recent events
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=EventName,AttributeValue=ConsoleLogin \
  --max-results 10

# Key events to monitor:
# ConsoleLogin — user logins (check for no MFA, unusual source IP)
# CreateUser / CreateAccessKey — new credentials created
# PutBucketPolicy — S3 bucket policy changes
# AuthorizeSecurityGroupIngress — firewall rule changes
# StopLogging — someone disabling CloudTrail (critical alert)
```

### Azure Monitor and GCP Audit Logs

```bash
# Azure Activity Log — query recent operations
az monitor activity-log list \
  --resource-group prod-rg \
  --start-time 2026-04-01 \
  --output table

# Azure diagnostic settings — send logs to Log Analytics
az monitor diagnostic-settings create \
  --name send-to-la \
  --resource /subscriptions/<sub-id>/resourceGroups/prod-rg \
  --workspace /subscriptions/<sub-id>/resourceGroups/prod-rg/providers/Microsoft.OperationalInsights/workspaces/prod-la \
  --logs '[{"category":"Administrative","enabled":true}]'

# GCP Audit Logs — query admin activity
gcloud logging read \
  'logName="projects/my-project/logs/cloudaudit.googleapis.com%2Factivity"
   AND protoPayload.methodName="SetIamPolicy"' \
  --limit=10 \
  --format=json

# GCP — export logs to BigQuery for analysis
gcloud logging sinks create audit-to-bq \
  bigquery.googleapis.com/projects/my-project/datasets/audit_logs \
  --log-filter='logName="projects/my-project/logs/cloudaudit.googleapis.com%2Factivity"'
```

## Container Security

### Image Scanning

```bash
# Scan container images for vulnerabilities

# AWS ECR image scanning (on push)
aws ecr put-image-scanning-configuration \
  --repository-name my-app \
  --image-scanning-configuration scanOnPush=true

# Get scan findings
aws ecr describe-image-scan-findings \
  --repository-name my-app \
  --image-id imageTag=latest

# Trivy — open-source scanner (works with any registry)
trivy image my-registry/my-app:latest
trivy image --severity HIGH,CRITICAL my-registry/my-app:latest

# Grype — open-source scanner
grype my-registry/my-app:latest

# Best practices for container images:
# - Use minimal base images (distroless, alpine)
# - Pin base image versions (no :latest in production)
# - Run as non-root user (USER directive in Dockerfile)
# - No secrets in image layers (use runtime injection)
# - Multi-stage builds to exclude build tools from final image
# - Scan in CI/CD pipeline; block deployment on critical CVEs
```

### Runtime Container Security

```bash
# Falco — open-source runtime security for containers and Kubernetes

# Example Falco rule: detect shell spawned in container
# - rule: Terminal shell in container
#   desc: Detect a shell being spawned in a container
#   condition: >
#     spawned_process and container and shell_procs
#     and not proc.pname in (cron, crond)
#   output: >
#     Shell spawned in container
#     (user=%user.name container=%container.name shell=%proc.name
#      parent=%proc.pname cmdline=%proc.cmdline)
#   priority: WARNING

# Pod Security Standards (Kubernetes)
# Restricted: no privilege escalation, no host namespaces, non-root
# Baseline: prevents known privilege escalations
# Privileged: unrestricted (only for system-level pods)

# Network policies — restrict pod-to-pod traffic
# kubectl apply -f network-policy.yaml
# apiVersion: networking.k8s.io/v1
# kind: NetworkPolicy
# spec:
#   podSelector:
#     matchLabels:
#       app: database
#   policyTypes: ["Ingress"]
#   ingress:
#     - from:
#         - podSelector:
#             matchLabels:
#               app: backend
#       ports:
#         - port: 5432
```

## Serverless Security

```bash
# Lambda function security best practices

# Minimum IAM permissions (use resource-based policies)
# Avoid: arn:aws:s3:::* (too broad)
# Use:   arn:aws:s3:::my-specific-bucket/prefix/*

# Environment variable encryption
aws lambda update-function-configuration \
  --function-name my-func \
  --kms-key-arn arn:aws:kms:us-east-1:123456789012:key/<key-id> \
  --environment "Variables={DB_HOST=prod-db.internal,DB_NAME=myapp}"

# VPC placement for private resource access
aws lambda update-function-configuration \
  --function-name my-func \
  --vpc-config SubnetIds=subnet-abc123,SecurityGroupIds=sg-abc123

# Reserved concurrency (prevent denial-of-wallet attacks)
aws lambda put-function-concurrency \
  --function-name my-func \
  --reserved-concurrent-executions 100

# Key serverless risks:
# - Over-privileged execution roles
# - Dependency vulnerabilities (node_modules, pip packages)
# - Event injection (untrusted input in triggers)
# - Denial of wallet (unlimited concurrency = unlimited cost)
# - Secrets in environment variables (use Secrets Manager instead)
# - Cold start timing attacks
```

## Cloud Compliance

### CIS Benchmarks

```bash
# AWS CIS Benchmark — automated checks

# Enable AWS Security Hub with CIS standard
aws securityhub enable-security-hub
aws securityhub batch-enable-standards \
  --standards-subscription-requests '[{
    "StandardsArn": "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.4.0"
  }]'

# Get CIS compliance findings
aws securityhub get-findings \
  --filters '{
    "GeneratorId": [{"Value": "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark", "Comparison": "PREFIX"}],
    "ComplianceStatus": [{"Value": "FAILED", "Comparison": "EQUALS"}]
  }'

# AWS Config rules — continuous compliance monitoring
aws configservice put-config-rule \
  --config-rule '{
    "ConfigRuleName": "s3-bucket-public-read-prohibited",
    "Source": {
      "Owner": "AWS",
      "SourceIdentifier": "S3_BUCKET_PUBLIC_READ_PROHIBITED"
    }
  }'

# Key CIS checks:
# 1.1  — Avoid use of root account
# 1.4  — Ensure MFA is enabled for root
# 1.10 — Ensure MFA is enabled for console access
# 2.1  — Ensure CloudTrail is enabled in all regions
# 2.6  — Ensure S3 bucket access logging is enabled
# 3.1  — Ensure CloudWatch log metric filter for unauthorized API calls
# 4.1  — Ensure no security groups allow ingress from 0.0.0.0/0 to port 22
```

## SASE Architecture

```
# Secure Access Service Edge — converged network + security

+-------------------+
| SASE Platform     |
| +------+ +------+ |
| | SD-WAN| | SWG  | |    SWG = Secure Web Gateway
| +------+ +------+ |    CASB = Cloud Access Security Broker
| +------+ +------+ |    ZTNA = Zero Trust Network Access
| | CASB | | ZTNA | |    FWaaS = Firewall as a Service
| +------+ +------+ |
| +------+          |
| |FWaaS |          |
| +------+          |
+-------------------+
        |
   Cloud-delivered
   PoP (Point of Presence)
        |
  +-----+-----+
  |           |
Users      Branches

# Key principles:
# - Identity-driven: access based on user identity, not network location
# - Cloud-native: security functions delivered from cloud PoPs
# - All edges: supports any user, any device, any location
# - Converged: networking and security in one platform
# - Examples: Cisco Umbrella + SD-WAN, Zscaler, Palo Alto Prisma SASE
```

## Cloud DLP

```bash
# Google Cloud DLP — scan for sensitive data

# Inspect content for PII
gcloud dlp inspect-content \
  --min-likelihood=LIKELY \
  --info-types="PHONE_NUMBER,EMAIL_ADDRESS,CREDIT_CARD_NUMBER" \
  --content="Contact me at john@example.com or 555-123-4567"

# AWS Macie — S3 data discovery and classification
aws macie2 enable-macie
aws macie2 create-classification-job \
  --job-type ONE_TIME \
  --s3-job-definition '{
    "bucketDefinitions": [{
      "accountId": "123456789012",
      "buckets": ["sensitive-data-bucket"]
    }]
  }' \
  --name "PII scan"

# DLP categories to scan for:
# - PII: names, addresses, phone numbers, SSN, DOB
# - Financial: credit card numbers, bank accounts, tax IDs
# - Healthcare: medical record numbers, diagnosis codes
# - Credentials: API keys, passwords, tokens, private keys
# - Custom patterns: internal identifiers, project codenames
```

## Tips

- Enable CloudTrail or equivalent audit logging in every cloud account from day one; it is the single most important security control.
- Use infrastructure-as-code (Terraform, CloudFormation) for all security configurations; manual console changes create drift and audit gaps.
- Never embed credentials in code or environment variables; use IAM roles, managed identities, or Workload Identity Federation.
- Apply least-privilege IAM policies; start with zero permissions and add only what is needed, using policy simulator to verify.
- Encrypt all data at rest using customer-managed keys (CMK) in production; provider-managed keys are acceptable for dev/test.
- Use VPC endpoints or private link for all AWS service access from private subnets to avoid data traversing the internet.
- Enable MFA on every human account; enforce via conditional access or IAM policies, not just documentation.
- Scan container images in CI/CD pipelines before they reach any registry; block images with critical or high CVEs.
- Use CSPM tools to continuously monitor for misconfigurations; point-in-time audits miss configuration drift.
- Implement network segmentation with security groups and NACLs in layers; security groups for instance-level, NACLs for subnet-level.
- Set up billing alerts and concurrency limits on serverless functions to prevent denial-of-wallet attacks.
- Use separate AWS accounts (or GCP projects, Azure subscriptions) for production, staging, and development to create hard IAM boundaries.

## See Also

- endpoint-security, security-operations, terraform, aws-cli, gcloud, azure-cli

## References

- [AWS Well-Architected Security Pillar](https://docs.aws.amazon.com/wellarchitected/latest/security-pillar/)
- [Azure Security Documentation](https://learn.microsoft.com/en-us/azure/security/)
- [GCP Security Best Practices](https://cloud.google.com/security/best-practices)
- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks)
- [NIST SP 800-144 — Guidelines on Security and Privacy in Public Cloud Computing](https://csrc.nist.gov/publications/detail/sp/800-144/final)
- [CSA Cloud Controls Matrix](https://cloudsecurityalliance.org/research/cloud-controls-matrix)
- [Cisco Umbrella SASE Documentation](https://docs.umbrella.com/)
- [Trivy Container Scanner](https://aquasecurity.github.io/trivy/)
- [Falco Runtime Security](https://falco.org/docs/)
