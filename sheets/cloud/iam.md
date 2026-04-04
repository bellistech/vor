# IAM (Identity and Access Management)

Cloud identity framework controlling who (principals) can do what (actions) on which resources, using policies, roles, and permissions across AWS, GCP, and Azure.

## Core Concepts

### Principals and identities

```bash
# AWS — list IAM users
aws iam list-users --output table

# AWS — list roles
aws iam list-roles --query 'Roles[].{Name:RoleName,Arn:Arn}' --output table

# AWS — get current caller identity
aws sts get-caller-identity

# GCP — list IAM members on a project
gcloud projects get-iam-policy my-project

# Azure — list role assignments
az role assignment list --output table
```

### Policy structure (AWS)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowS3Read",
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
          "aws:SourceIp": "203.0.113.0/24"
        }
      }
    }
  ]
}
```

## AWS IAM

### Users and groups

```bash
# Create a user
aws iam create-user --user-name deploy-bot

# Add user to a group
aws iam add-user-to-group --user-name deploy-bot --group-name developers

# Create access keys (store securely)
aws iam create-access-key --user-name deploy-bot

# Enable MFA for a user
aws iam enable-mfa-device \
  --user-name deploy-bot \
  --serial-number arn:aws:iam::123456789012:mfa/deploy-bot \
  --authentication-code1 123456 \
  --authentication-code2 789012
```

### Roles and trust policies

```bash
# Create a role with trust policy
aws iam create-role \
  --role-name lambda-exec \
  --assume-role-policy-document file://trust-policy.json

# Attach managed policy to role
aws iam attach-role-policy \
  --role-name lambda-exec \
  --policy-arn arn:aws:iam::aws:policy/AWSLambdaBasicExecutionRole

# Assume a role (get temporary credentials)
aws sts assume-role \
  --role-arn arn:aws:iam::123456789012:role/admin-role \
  --role-session-name my-session \
  --duration-seconds 3600
```

### Inline vs managed policies

```bash
# Create a managed policy
aws iam create-policy \
  --policy-name s3-read-only \
  --policy-document file://s3-readonly.json

# Put an inline policy on a user (avoid — prefer managed)
aws iam put-user-policy \
  --user-name deploy-bot \
  --policy-name emergency-access \
  --policy-document file://inline.json

# List attached policies for a role
aws iam list-attached-role-policies --role-name lambda-exec
```

### Policy boundaries and SCPs

```bash
# Set a permissions boundary on a user
aws iam put-user-permissions-boundary \
  --user-name deploy-bot \
  --permissions-boundary arn:aws:iam::123456789012:policy/boundary

# List SCPs in an organization
aws organizations list-policies --filter SERVICE_CONTROL_POLICY

# Attach SCP to an OU
aws organizations attach-policy \
  --policy-id p-abc123 \
  --target-id ou-root-example
```

## GCP IAM

### Roles and bindings

```bash
# Grant a role to a member
gcloud projects add-iam-policy-binding my-project \
  --member="user:alice@example.com" \
  --role="roles/storage.objectViewer"

# List predefined roles
gcloud iam roles list --filter="name:roles/compute"

# Create a custom role
gcloud iam roles create customEditor \
  --project=my-project \
  --title="Custom Editor" \
  --permissions="storage.objects.get,storage.objects.list"

# Test IAM permissions
gcloud asset analyze-iam-policy \
  --organization=123456 \
  --identity="user:alice@example.com"
```

### Service accounts

```bash
# Create a service account
gcloud iam service-accounts create my-sa \
  --display-name="My Service Account"

# Generate a key file
gcloud iam service-accounts keys create key.json \
  --iam-account=my-sa@my-project.iam.gserviceaccount.com

# Impersonate a service account
gcloud auth print-access-token \
  --impersonate-service-account=my-sa@my-project.iam.gserviceaccount.com

# Grant workload identity to a k8s service account
gcloud iam service-accounts add-iam-policy-binding \
  my-sa@my-project.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:my-project.svc.id.goog[namespace/ksa-name]"
```

## Azure IAM

### Role assignments

```bash
# Assign a built-in role
az role assignment create \
  --assignee alice@example.com \
  --role "Reader" \
  --scope /subscriptions/sub-id/resourceGroups/my-rg

# List role definitions
az role definition list --output table

# Create a custom role
az role definition create --role-definition @custom-role.json

# List assignments for a resource group
az role assignment list \
  --resource-group my-rg --output table
```

### Managed identities

```bash
# Create a user-assigned managed identity
az identity create \
  --name my-identity \
  --resource-group my-rg

# Assign managed identity to a VM
az vm identity assign \
  --name my-vm \
  --resource-group my-rg \
  --identities my-identity

# System-assigned identity — enable on a web app
az webapp identity assign --name my-app --resource-group my-rg
```

## RBAC vs ABAC

### RBAC (Role-Based Access Control)

```bash
# AWS — traditional RBAC via groups
aws iam create-group --group-name developers
aws iam attach-group-policy \
  --group-name developers \
  --policy-arn arn:aws:iam::aws:policy/PowerUserAccess
aws iam add-user-to-group --user-name alice --group-name developers
```

### ABAC (Attribute-Based Access Control)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "ec2:*",
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "aws:ResourceTag/Department": "${aws:PrincipalTag/Department}"
        }
      }
    }
  ]
}
```

## Least Privilege Patterns

### Access Analyzer

```bash
# AWS — enable IAM Access Analyzer
aws accessanalyzer create-analyzer \
  --analyzer-name my-analyzer \
  --type ACCOUNT

# List findings (overly permissive resources)
aws accessanalyzer list-findings --analyzer-arn arn:aws:...

# Generate a policy from CloudTrail activity
aws accessanalyzer start-policy-generation \
  --policy-generation-details file://policy-gen.json
```

### Audit and review

```bash
# AWS — generate credential report
aws iam generate-credential-report
aws iam get-credential-report --output text \
  --query 'Content' | base64 -d > cred-report.csv

# Find unused roles (last used > 90 days)
aws iam get-role --role-name my-role \
  --query 'Role.RoleLastUsed.LastUsedDate'

# GCP — IAM recommender
gcloud recommender recommendations list \
  --project=my-project \
  --recommender=google.iam.policy.Recommender \
  --location=global
```

## Tips

- Always prefer managed policies over inline policies for reuse and version tracking.
- Use conditions (IP, time, MFA, tags) to narrow permissions beyond simple Allow/Deny.
- Enable MFA on all human accounts and require it in trust policies with `aws:MultiFactorAuthPresent`.
- Use permissions boundaries to limit the maximum permissions a role or user can ever have.
- Rotate access keys regularly and prefer temporary credentials via STS AssumeRole.
- Tag principals and resources consistently to enable ABAC at scale.
- Run IAM Access Analyzer continuously to detect unintended external access.
- Use service control policies (SCPs) at the organization level to set guardrails across all accounts.
- Generate credential reports monthly and disable unused keys older than 90 days.
- Prefer workload identity federation (OIDC) over long-lived service account keys.
- Use separate accounts/projects per environment (dev/staging/prod) for blast radius containment.
- Review the GCP IAM Recommender and AWS Access Advisor to right-size permissions.

## See Also

aws-cli, gcloud, azure-cli, vpc, sealed-secrets, sops

## References

- [AWS IAM User Guide](https://docs.aws.amazon.com/IAM/latest/UserGuide/)
- [AWS IAM Policy Reference](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies.html)
- [AWS IAM Access Analyzer](https://docs.aws.amazon.com/IAM/latest/UserGuide/what-is-access-analyzer.html)
- [GCP IAM Overview](https://cloud.google.com/iam/docs/overview)
- [GCP Understanding Roles](https://cloud.google.com/iam/docs/understanding-roles)
- [Azure RBAC Documentation](https://learn.microsoft.com/en-us/azure/role-based-access-control/)
- [Azure Managed Identities](https://learn.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/)
- [NIST RBAC Model (SP 800-162)](https://csrc.nist.gov/publications/detail/sp/800-162/final)
