# AWS Data Protection & Encryption (SAA-C03 Task 1.3)

> Determine appropriate data security controls: KMS key types and policies, envelope encryption, S3 encryption modes, ACM certificates, backup/replication/retention, Object Lock, and compliance alignment.

## KMS Fundamentals

```
# KMS key (formerly CMK) = metadata + reference to key material that
# NEVER leaves KMS unencrypted. All crypto happens inside KMS/HSMs.
#
# Key types:
# AWS managed keys  (aws/s3, aws/ebs, ...) — created/rotated (yearly)
#   by AWS; no key policy control; free; cannot be used cross-account
# Customer managed keys (CMK) — you control policy, rotation, grants,
#   aliases; 1 USD/month + API usage; cross-account capable — the
#   answer whenever the question says "control/audit the key" or
#   "share encrypted data with another account"
# AWS owned keys — invisible, shared fleet keys (e.g. default DynamoDB)
#
# Key material origin:
#   KMS-generated (default) | External (BYOK import; you manage expiry,
#   no auto-rotation) | CloudHSM-backed (custom key store) |
#   External key store (XKS — keys stay in YOUR HSM outside AWS)
#
# Symmetric (AES-256-GCM, default; encrypt/decrypt never leaves KMS)
# Asymmetric (RSA/ECC — sign/verify or encrypt/decrypt; public key
#   downloadable) — "verify signatures outside AWS" → asymmetric
# HMAC keys — generate/verify MACs
#
# Rotation:
# AWS managed        — every year, automatic, can't change
# Customer managed   — optional automatic yearly (keeps old versions
#   to decrypt old data transparently); imported material = manual
#   rotation only (new key + re-point alias)
# Manual rotation via alias re-pointing — apps reference alias, never
#   the key id, so rotation is a no-op for callers
```

```bash
aws kms create-key --description "app data key"
aws kms create-alias --alias-name alias/app --target-key-id <key-id>
aws kms enable-key-rotation --key-id <key-id>            # yearly auto
aws kms encrypt --key-id alias/app --plaintext fileb://s.txt \
  --query CiphertextBlob --output text
aws kms generate-data-key --key-id alias/app --key-spec AES_256
aws kms schedule-key-deletion --key-id <key-id> \
  --pending-window-in-days 30      # 7–30 day mandatory waiting period
```

## Envelope Encryption (How Everything Encrypts)

```
# KMS symmetric API caps payloads at 4 KB → services encrypt bulk data
# with envelope encryption:
# 1. GenerateDataKey → returns plaintext DEK + DEK encrypted under CMK
# 2. Encrypt data locally with plaintext DEK (AES-256), discard it
# 3. Store encrypted DEK alongside ciphertext
# 4. Decrypt path: kms:Decrypt on encrypted DEK → plaintext DEK → data
# Benefits: one KMS call per object (cost/latency), key never in transit
# with data, per-object DEKs. S3-SSE-KMS, EBS, RDS all do exactly this.
# S3 Bucket Keys: per-bucket intermediate key slashes KMS request costs
# (fewer GenerateDataKey calls) — "reduce KMS costs for S3" answer
```

## Key Policies, Grants, Cross-Account

```
# EVERY KMS key has exactly one key policy (resource policy, mandatory).
# Default policy: "Principal": {"AWS":"arn:aws:iam::ACCOUNT:root"} allow
#   * → delegates control to IAM policies in the account.
# Remove that statement and IAM policies alone CANNOT grant access —
#   locked-out keys are a real support case; exam: key policy is the
#   primary control, IAM is secondary.
# Cross-account use: key policy allows external account +
#   external identity policy allows kms:Decrypt/Encrypt → both sides.
# Grants — programmatic, temporary, single-key permissions used by AWS
#   services (EBS uses grants to decrypt volumes for EC2).
# ViaService condition — restrict a key to use through one service:
#   "kms:ViaService":"s3.eu-west-1.amazonaws.com"
```

## Encryption at Rest, Service by Service

```
# S3 — all new objects encrypted by default (SSE-S3) since 2023
#   SSE-S3  (AES256)   — S3-managed keys, free, no key control
#   SSE-KMS (aws:kms)  — CMK control + CloudTrail audit of key use;
#                        Bucket Keys to cut cost; API quota applies
#   DSSE-KMS           — double encryption (two independent layers)
#   SSE-C              — customer supplies key per request over HTTPS;
#                        AWS keeps nothing; you lose key = you lose data
#   Client-side        — encrypt before upload (S3 Encryption Client)
#   Enforce KMS: bucket policy Deny unless
#     "s3:x-amz-server-side-encryption":"aws:kms"
#
# EBS — set account-level "encryption by default" per Region; encrypts
#   volume, snapshots, and instance↔volume traffic. Cannot encrypt an
#   existing unencrypted volume in place: snapshot → copy-with-
#   encryption → new volume (same trick changes the key).
# RDS/Aurora — encrypt at CREATE time only; existing unencrypted DB:
#   snapshot → copy snapshot encrypted → restore. Encrypted primary =
#   encrypted replicas (and vice versa; can't mix).
# DynamoDB — always encrypted; choose AWS owned (free) / AWS managed /
#   customer managed key.
# EFS/FSx — enable at creation (EFS in-transit via TLS mount option)
# Redshift/ElastiCache/SQS/SNS/Kinesis — KMS options throughout
```

## Encryption in Transit + ACM

```
# TLS everywhere: ALB/NLB/CloudFront/API Gateway terminate TLS with
# ACM certificates.
# ACM public certs — FREE, auto-renewing (DNS validation via Route 53
#   CNAME = zero-touch renewal; email validation = manual).
# Regional rule: cert must live in the SAME Region as the ALB/NLB/API
#   Gateway. CloudFront certs MUST be in us-east-1. ACM certs cannot
#   be exported (except private CA certs) — no downloading to EC2.
# ACM Private CA — issue internal certs (mTLS, internal domains); paid.
# IAM server certs — legacy store, only for Regions without ACM.
# End-to-end encryption behind an ALB: ALB re-encrypts to targets
#   (HTTPS target group; self-signed certs on targets are accepted).
# NLB TLS passthrough (TCP listener) when targets must hold the cert /
#   client-cert auth (mTLS terminates on the instance).
# S3: enforce TLS with bucket policy Deny on aws:SecureTransport=false.
# CloudFront: Viewer Protocol Policy redirect-to-HTTPS + Origin
#   Protocol Policy https-only.
```

## S3 Data Protection Controls

```
# Block Public Access — 4 switches at account AND bucket level; leave
#   ON; overrides any ACL/policy that would expose data ("ensure no
#   bucket can ever be made public" → account-level BPA)
# Versioning — keeps every object version; delete = delete marker;
#   protects against overwrite/delete; suspend ≠ remove old versions
# MFA Delete — require MFA to delete versions / change versioning
#   state; root + CLI only to enable — "protect against accidental
#   permanent deletion, maximum control" answer
# Object Lock (WORM) — needs versioning; modes:
#   Governance — special permission can override/shorten retention
#   Compliance — NOBODY (not even root) can delete/shorten until expiry
#   Legal hold — indefinite lock independent of retention dates
#   Trigger: "regulatory WORM storage", "ransomware-proof backups"
# Glacier Vault Lock — the WORM equivalent for classic Glacier vaults
# Access Points — per-application entry points with their own policies
#   (simplify huge shared buckets); Object Lambda transforms on read
#   (redact PII for one consumer without duplicating data)
# Pre-signed URLs — time-limited GET/PUT with the signer's permissions
#   ("let users download/upload one object briefly, no credentials")
# Replication: SRR (same Region) / CRR (cross Region) — needs
#   versioning both sides; new objects only unless S3 Batch Replication
#   backfills; can replicate to another ACCOUNT (ownership override)
#   for tamper-proof log archives; RTC (Replication Time Control) = 15
#   min SLA. Delete markers optionally replicated; version deletes never.
```

## Backup, Retention, Recovery

```
# AWS Backup — central, policy-based backup across services: EBS, EC2,
#   RDS/Aurora, DynamoDB, EFS, FSx, Storage Gateway, S3, Neptune,
#   DocumentDB, VMware. Backup plans (schedule + lifecycle to cold
#   storage + retention), cross-Region AND cross-account copies (via
#   Organizations), Backup Vault Lock (WORM vaults — ransomware answer),
#   legal holds, restore testing plans.
#   Trigger: "centralize/standardize backups", "prove compliance with
#   backup policies org-wide", "immutable backups"
# Service-native alternatives:
#   EBS snapshots (incremental, to S3 internally) + Data Lifecycle
#     Manager (DLM) for scheduling — EBS-only shops
#   RDS automated backups (PITR within retention 0–35 days) vs manual
#     snapshots (keep forever)
#   DynamoDB PITR (35 days) + on-demand backups
#   S3 versioning + lifecycle + replication as "backup"
# RPO/RTO drive the choice — DR strategy table lives in
#   cs aws-high-availability
```

## Data Classification & Governance

```
# Classify: public / internal / confidential / restricted → map each
#   class to controls (key type, BPA, Object Lock, logging, retention)
# Discover what you actually store: Macie (PII in S3) →
#   cs aws-network-security
# Tag-based governance: mandatory tags via Organizations Tag Policies +
#   SCP deny on missing tags; lifecycle by tag; access by tag (ABAC)
# Audit key usage: every KMS call lands in CloudTrail ("prove who
#   decrypted what, when" → CloudTrail + customer managed keys)
# Residency: SCP aws:RequestedRegion deny + KMS keys are Regional
#   (multi-Region keys replicate material for cross-Region decrypt
#   without re-encryption — DR + global tables use case)
# Compliance evidence: AWS Artifact (download SOC/ISO/PCI reports),
#   Audit Manager (continuous evidence collection against frameworks)
```

## CloudHSM vs KMS

```
# CloudHSM — single-tenant FIPS 140-2 Level 3 HSM cluster in your VPC;
#   you own the crypto users; AWS has NO access to keys; PKCS#11/JCE/
#   CNG interfaces; use as KMS custom key store or standalone
# KMS — multi-tenant, FIPS 140-2 Level 2 (some Level 3 regions),
#   deeply integrated with every AWS service
# Choose CloudHSM when: "FIPS 140-2 Level 3 required", "keys must be
#   in dedicated hardware only we control", "offload TLS/Oracle TDE",
#   "SQL Server/Oracle native encryption with own HSM"
# Otherwise KMS. Hybrid: KMS custom key store backed by CloudHSM =
#   KMS API + dedicated hardware.
```

## Exam Traps

```
# Existing unencrypted EBS/RDS cannot be encrypted in place — the
#   snapshot-copy-encrypt-restore dance is the answer every time
# SSE-KMS adds CloudTrail audit + key control; SSE-S3 has neither
# S3 Bucket Keys = the "reduce KMS request costs" answer
# CloudFront certificate Region = us-east-1, always
# ACM public certs auto-renew ONLY with DNS validation reachable;
#   email validation expires on silent inboxes
# Object Lock Compliance mode is irreversible — even root waits
# MFA Delete ≠ Object Lock: MFA-Delete gates deletes behind MFA;
#   Object Lock is WORM retention
# Cross-account encrypted snapshot sharing: can't share snapshots
#   encrypted with DEFAULT aws/ebs key — re-encrypt with a customer
#   managed key, share snapshot + grant kms:Decrypt in key policy
# Key deletion has a 7–30 day pending window; data encrypted under a
#   deleted key is GONE — disable keys instead when unsure
# "Company must manage key material outside AWS" → BYOK import or XKS
# "Audit every use of the encryption key" → customer managed KMS key
#   + CloudTrail (aws-managed keys give less policy control)
# Replication is not backup: deletes/corruption replicate too —
#   versioning + Object Lock/Vault Lock is the immutability answer
```

## See Also

aws-saa-c03, aws-iam, aws-network-security, aws-storage, aws-high-availability, aws-monitoring-governance, s3, cloud-security, cryptography, pki, tls, vault, backup

## References

- [AWS KMS Developer Guide](https://docs.aws.amazon.com/kms/latest/developerguide/overview.html)
- [KMS key policies](https://docs.aws.amazon.com/kms/latest/developerguide/key-policies.html)
- [Envelope encryption concepts](https://docs.aws.amazon.com/kms/latest/developerguide/concepts.html#enveloping)
- [S3 server-side encryption options](https://docs.aws.amazon.com/AmazonS3/latest/userguide/serv-side-encryption.html)
- [S3 Object Lock](https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-lock.html)
- [ACM User Guide](https://docs.aws.amazon.com/acm/latest/userguide/acm-overview.html)
- [AWS Backup Developer Guide](https://docs.aws.amazon.com/aws-backup/latest/devguide/whatisbackup.html)
- [CloudHSM User Guide](https://docs.aws.amazon.com/cloudhsm/latest/userguide/introduction.html)
- [EBS encryption](https://docs.aws.amazon.com/ebs/latest/userguide/ebs-encryption.html)
