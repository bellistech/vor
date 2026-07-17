# AWS IAM & Secure Access (SAA-C03 Task 1.1)

> Design secure access to AWS resources: IAM identities, policy types and evaluation logic, STS and role assumption, federation, IAM Identity Center, Organizations/SCPs, and multi-account access patterns.

## Identity Model

```
# Account root user
# - Full, unrestrictable access; cannot be limited by IAM policy or SCP
#   (SCPs DO apply to member-account root... only in an Organization —
#    the management account root is never restricted by SCPs)
# - Use ONLY for: closing account, changing support plan, some billing
#   tasks, restoring accidentally-removed admin access
# - Lock down: enable MFA (hardware key ideal), delete root access keys

# IAM user      — long-term identity with password and/or access keys
#                 (exam: an anti-pattern for humans; prefer federation)
# IAM group     — collection of users; policies attach to groups
#                 (groups cannot be nested; groups are not principals —
#                  you cannot reference a group in a resource policy)
# IAM role      — assumable identity with temporary credentials via STS;
#                 no password, no long-term keys; has a trust policy
#                 (who can assume) + permission policies (what it can do)
# Service-linked role — role owned/managed by an AWS service
#                 (e.g. AWSServiceRoleForAutoScaling)
# Instance profile — container that passes a role to an EC2 instance

# Federated identity — external IdP user mapped into a role:
#   SAML 2.0 (AD FS, Okta), OIDC (Google, GitHub Actions), Cognito
```

## Policy Types (Know All Six)

```
# 1. Identity-based  — attach to user/group/role (managed or inline)
# 2. Resource-based  — attach to resource (S3 bucket policy, SQS queue
#      policy, KMS key policy, Lambda resource policy, SNS topic policy);
#      only these support cross-account grants WITHOUT role assumption;
#      Principal element is required
# 3. Permission boundary — identity-based policy that sets the MAXIMUM
#      permissions an IAM user/role can have; grants nothing by itself
# 4. Service control policy (SCP) — Organizations guardrail; sets max
#      permissions for accounts/OUs; grants nothing; doesn't affect
#      resource-based policy access granted TO other accounts;
#      doesn't apply to the management account
# 5. Session policy — passed at AssumeRole time; further limits the
#      session's effective permissions
# 6. ACL — legacy (S3 object ACLs); disable with S3 Object Ownership =
#      "Bucket owner enforced" (default for new buckets since 2023)

# Effective permission = intersection of all applicable layers:
#   SCP ∩ permission boundary ∩ session policy ∩
#   (identity policy ∪ resource policy†)
# † within the SAME account, identity OR resource policy allow suffices;
#   cross-account requires an Allow on BOTH sides
```

## Policy Document Anatomy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowBucketRW",
      "Effect": "Allow",
      "Action": ["s3:GetObject", "s3:PutObject"],
      "Resource": "arn:aws:s3:::app-bucket/team/${aws:username}/*",
      "Condition": {
        "Bool": {"aws:MultiFactorAuthPresent": "true"},
        "IpAddress": {"aws:SourceIp": "203.0.113.0/24"},
        "StringEquals": {"aws:PrincipalOrgID": "o-a1b2c3d4e5"}
      }
    }
  ]
}
```

```
# Version — always "2012-10-17" (omitting it breaks policy variables)
# NotAction / NotResource — match everything EXCEPT listed (common in
#   deny-all-but patterns; pair with "Effect":"Deny" carefully)
# Wildcards: s3:Get*  arn:aws:s3:::bucket/*
# Key condition keys for the exam:
#   aws:SourceIp             — caller IP (public; useless behind VPC endpoint)
#   aws:SourceVpce           — restrict to a specific VPC endpoint
#   aws:PrincipalOrgID       — any principal in my Organization
#   aws:MultiFactorAuthPresent — MFA enforced
#   aws:RequestedRegion      — restrict where resources are created
#   aws:SecureTransport      — deny non-TLS (S3 bucket policy staple)
#   sts:ExternalId           — third-party confused-deputy protection
#   aws:ResourceTag/  aws:PrincipalTag/  — ABAC (attribute-based access)
```

## Policy Evaluation Logic (Exam Favorite)

```
# Within one account, for a request:
# 1. Default: implicit DENY (everything starts denied)
# 2. Evaluate ALL applicable policies
# 3. Explicit Deny anywhere → DENIED (nothing overrides explicit deny)
# 4. SCP allows? If not → DENIED
# 5. Resource-based policy allows the principal → ALLOWED (same account,
#    even if identity policy is silent)
# 6. Identity policy allows → check permission boundary:
#    boundary exists and doesn't allow → DENIED (implicit)
# 7. Session policy (if present) must also allow
# 8. Otherwise → implicit DENY
#
# Cross-account: requester's identity policy must allow the call AND
# the resource policy must allow the external principal. No resource
# policy = no cross-account access (role assumption is the alternative).
#
# Order of loss: explicit deny > SCP deny > boundary miss > implicit deny
```

## STS and Role Assumption

```bash
# Assume a role (returns temporary creds: AccessKey, Secret, Token)
aws sts assume-role \
  --role-arn arn:aws:iam::222222222222:role/DeployRole \
  --role-session-name deploy-$(date +%s) \
  --duration-seconds 3600          # 900s–max session (default max 1h,
                                   # role setting raises to 12h)

aws sts get-caller-identity        # who am I (account, ARN, user id)
aws sts decode-authorization-message --encoded-message <blob>  # decode
                                   # "not authorized" encoded errors
```

```
# STS API family:
# AssumeRole              — role in same/other account (MFA optional)
# AssumeRoleWithSAML      — enterprise IdP federation (AD FS, Okta)
# AssumeRoleWithWebIdentity — OIDC (Cognito, GitHub OIDC, Google)
# GetSessionToken         — MFA-protected temporary creds for IAM user
# GetFederationToken      — legacy federation (rarely correct answer)

# Role chaining: role→role assumption caps sessions at 1 hour
# Credentials on EC2: instance profile → metadata service (IMDSv2!)
#   http://169.254.169.254/latest/meta-data/iam/security-credentials/
# Enforce IMDSv2 (session-token metadata) to kill SSRF credential theft:
```

```bash
aws ec2 modify-instance-metadata-options --instance-id i-0abc \
  --http-tokens required --http-endpoint enabled
```

## Cross-Account Access Patterns

```
# Pattern A — role in the target account (the default answer):
# 1. Account B creates role with trust policy naming Account A
# 2. Account A identity policy allows sts:AssumeRole on that role ARN
# 3. Caller assumes role, works inside B with temporary creds
#
# Trust policy (in account B):
# {"Effect":"Allow",
#  "Principal":{"AWS":"arn:aws:iam::111111111111:root"},
#  "Action":"sts:AssumeRole",
#  "Condition":{"StringEquals":{"sts:ExternalId":"vendor-42"}}}
# ("root" in a trust policy = "any principal in that account that has
#  been granted sts:AssumeRole" — it does NOT mean only the root user)
#
# Pattern B — resource-based policy (S3, SQS, SNS, KMS, Lambda):
#   grant the foreign account/principal directly; caller keeps their
#   own identity (advantage: no credential switch, can access home
#   account resources simultaneously)
#
# External ID — REQUIRED answer whenever a third party (SaaS vendor)
#   assumes a role in your account: prevents the confused deputy
#   (vendor tricked into using their access on the wrong customer)
#
# aws:PrincipalOrgID in a resource policy = "any account in my org" —
#   the scalable alternative to listing 50 account IDs
```

## IAM Identity Center (Successor to AWS SSO)

```
# One login → all accounts in the Organization + CLI/SDK sessions
# Identity sources: built-in directory, Active Directory (via AD
#   Connector or AWS Managed Microsoft AD), or external SAML IdP
#   (Okta, Entra ID) with SCIM user provisioning
# Permission sets → provisioned as roles into member accounts
# Exam triggers: "workforce users", "single sign-on to multiple AWS
#   accounts", "centrally manage access", "AD integration for console"
# vs Cognito: Identity Center = workforce (employees);
#             Cognito = customers (app sign-up/sign-in) — cs aws-network-security
# Directory Service options:
#   AWS Managed Microsoft AD — real AD in AWS, trust with on-prem AD
#   AD Connector             — proxy to on-prem AD (no AD data in cloud)
#   Simple AD                — Samba-based, small orgs, no trust support
```

## Organizations and SCPs

```
# AWS Organizations: management account + OUs + member accounts
# - Consolidated billing (one bill, volume-discount + RI/SP sharing)
# - SCPs, tag policies, backup policies, AI-services opt-out policies
# - Trusted access for services (Config, GuardDuty, ... org-wide)
# - Account creation API (vend accounts programmatically)
#
# SCP mechanics:
# - FullAWSAccess attached by default (allow *); strategy choices:
#     deny-list (keep FullAWSAccess, add targeted Denies)  ← common
#     allow-list (remove FullAWSAccess, explicitly allow)  ← strict
# - Inheritance: effective SCP = intersection along the OU path
# - Never affects: management account, service-linked roles,
#   resource-based policy grants to principals outside the account
#
# Classic exam SCPs:
# deny leaving the org, deny region use outside eu-west-1 + us-east-1
# (allow global services via NotAction), deny root user actions,
# deny disabling CloudTrail/GuardDuty
#
# Control Tower — opinionated landing zone ON TOP of Organizations:
#   Account Factory, guardrails (preventive = SCP, detective = Config),
#   dashboard. Exam trigger: "set up multi-account environment with
#   best-practice guardrails quickly" → Control Tower
# RAM (Resource Access Manager) — share resources (subnets, TGW,
#   Route 53 resolver rules, License Manager configs) across accounts
#   without duplication; "share a VPC subnet with another account" → RAM
```

## ABAC vs RBAC

```
# RBAC — one policy/role per job function; N teams × M projects = policy
#        explosion; fine for small orgs
# ABAC — tag principals and resources; one policy compares tags:
#   "Condition":{"StringEquals":
#     {"aws:ResourceTag/project":"${aws:PrincipalTag/project}"}}
# Exam trigger: "minimize number of policies as teams/projects grow",
# "new projects get access automatically without policy edits" → ABAC
```

## Operational Commands

```bash
aws iam create-role --role-name AppRole \
  --assume-role-policy-document file://trust.json
aws iam attach-role-policy --role-name AppRole \
  --policy-arn arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess
aws iam put-role-permissions-boundary --role-name AppRole \
  --permissions-boundary arn:aws:iam::111111111111:policy/DevBoundary

aws iam create-policy --policy-name TeamPolicy \
  --policy-document file://policy.json
aws iam create-policy-version --policy-arn arn:... \
  --policy-document file://v2.json --set-as-default   # 5 versions max

# Credential hygiene
aws iam generate-credential-report && aws iam get-credential-report \
  --query Content --output text | base64 -d > creds.csv   # key age, MFA
aws iam list-access-keys --user-name dev1
aws iam update-access-key --user-name dev1 \
  --access-key-id AKIA... --status Inactive              # rotate: create
                                                          # new → switch →
                                                          # inactive → delete
# Test policies before deploying
aws iam simulate-principal-policy \
  --policy-source-arn arn:aws:iam::111111111111:role/AppRole \
  --action-names s3:GetObject \
  --resource-arns arn:aws:s3:::app-bucket/file.txt
# Access Analyzer — finds resources shared outside the account/org,
# generates least-privilege policies from CloudTrail activity
aws accessanalyzer create-analyzer --analyzer-name org --type ORGANIZATION
aws accessanalyzer start-policy-generation ...
# Last-accessed data — prune unused permissions
aws iam generate-service-last-accessed-details --arn <role-arn>
```

## Exam Traps

```
# "IAM user with access keys on EC2" — always wrong; answer = IAM role
#   via instance profile
# "Share credentials between accounts" — never; assume role cross-account
# Groups are not principals: can't appear in trust or resource policies
# Explicit Deny beats everything, including resource-policy Allows
# Permission boundary without identity policy Allow = still no access
#   (boundary is a ceiling, not a grant)
# SCP does not grant; a user in a member account still needs IAM allows
# Management account is immune to SCPs — auditors' finding, exam trap
# Resource policies survive SCPs when the accessor is OUTSIDE the org?
#   No: SCPs restrict PRINCIPALS in member accounts; an external
#   principal accessing a member-account bucket is governed by the
#   bucket policy + their own account's controls, not your SCP
# Root user tasks (only root can): close account, change account name/
#   email, restore IAM admin, some tax/billing settings, S3 MFA-Delete
#   enable via CLI, deregister as seller
# AssumeRoleWithWebIdentity vs Cognito: mobile app at scale → Cognito
#   identity pools (handles anonymous, credential caching); bare OIDC
#   federation (GitHub Actions → AWS) → AssumeRoleWithWebIdentity + OIDC
#   provider, no Cognito
# MFA enforcement = Condition aws:MultiFactorAuthPresent on sensitive
#   actions; hardware MFA for root is the "most secure" option
# Access keys in code/repo → answer: roles, or Secrets Manager for
#   third-party keys; never environment-variable long-term keys
# "Audit who can access resource X externally" → IAM Access Analyzer
# "Generate policy from actual usage" → Access Analyzer policy generation
# "Which user made this API call" → CloudTrail (cs aws-monitoring-governance)
```

## See Also

aws-saa-c03, aws-network-security, aws-data-protection, aws-monitoring-governance, iam, cloud-security, identity-management, access-control-models, saml, oidc, zero-trust

## References

- [IAM User Guide](https://docs.aws.amazon.com/IAM/latest/UserGuide/introduction.html)
- [Policy evaluation logic](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_evaluation-logic.html)
- [IAM JSON policy element reference](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements.html)
- [STS API reference](https://docs.aws.amazon.com/STS/latest/APIReference/welcome.html)
- [Organizations SCPs](https://docs.aws.amazon.com/organizations/latest/userguide/orgs_manage_policies_scps.html)
- [IAM Identity Center User Guide](https://docs.aws.amazon.com/singlesignon/latest/userguide/what-is.html)
- [The confused deputy problem](https://docs.aws.amazon.com/IAM/latest/UserGuide/confused-deputy.html)
- [IAM Access Analyzer](https://docs.aws.amazon.com/IAM/latest/UserGuide/what-is-access-analyzer.html)
