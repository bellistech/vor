# Access Control Models

> Authorization frameworks governing how subjects (users, processes) access objects (files, resources) based on policies, roles, attributes, or relationships.

## Core Models Overview

```
Model    Full Name                     Decision Basis
─────    ─────────                     ──────────────
DAC      Discretionary Access Control  Owner-defined permissions
MAC      Mandatory Access Control      Security labels/clearances
RBAC     Role-Based Access Control     Assigned roles
ABAC     Attribute-Based Access Control Subject/object/env attributes
RuBAC    Rule-Based Access Control     Predefined rules (firewalls)
ReBAC    Relationship-Based AC         Object-relationship graphs
```

## Comparison Matrix

```
Feature          DAC     MAC     RBAC    ABAC    ReBAC
─────────        ───     ───     ────    ────    ─────
Admin overhead   Low     High    Medium  High    Medium
Flexibility      High    Low     Medium  Very    Very
                                        High    High
Scalability      Low     Medium  High    High    High
Granularity      Medium  High    Medium  Very    High
                                        High
Least privilege  Weak    Strong  Good    Best    Good
Central control  No      Yes     Yes     Yes     Yes
Typical use      Files   Military Enterprise  Cloud   Social
                         /Gov    /IT     /API    /Collab
```

## DAC — Discretionary Access Control

```
# Owner decides who can access their objects
# Common in Unix/Linux file systems and Windows NTFS

# Unix file permissions (owner/group/others)
chmod 750 /data/report.txt    # rwxr-x---
chown alice:finance report.txt

# ACL (Access Control List) on Linux
setfacl -m u:bob:rx /data/report.txt
setfacl -m g:auditors:r /data/report.txt
getfacl /data/report.txt

# Windows NTFS ACL
icacls C:\Data\report.txt /grant Bob:(R,RX)
icacls C:\Data\report.txt /deny Guest:(F)

# DAC Weaknesses
# - Owner can delegate access without oversight
# - Vulnerable to trojan horse (malware inherits user's permissions)
# - No central policy enforcement
# - Confused deputy problem
```

## MAC — Mandatory Access Control

```
# System-enforced labels — users cannot override
# Labels: classification (Top Secret > Secret > Confidential > Unclassified)
# Compartments: {NATO, CRYPTO, NUCLEAR}

# Bell-LaPadula Model (Confidentiality)
# - Simple Security Rule: No Read Up (subject cannot read higher classification)
# - Star (*) Property: No Write Down (subject cannot write to lower classification)
# - Strong Star Property: Read/write only at own level

# Biba Model (Integrity)
# - Simple Integrity Axiom: No Read Down (don't read less trustworthy data)
# - Star Integrity Axiom: No Write Up (don't corrupt higher integrity data)
# - Invocation Property: Cannot call higher integrity subjects

# SELinux (MAC on Linux)
# Type Enforcement
semanage fcontext -a -t httpd_sys_content_t "/web(/.*)?"
restorecon -Rv /web
# View labels
ls -Z /web
# ps -Z to see process labels
ps -eZ | grep httpd
# Booleans for policy tuning
setsebool -P httpd_can_network_connect on

# AppArmor (MAC on Linux — path-based)
aa-enforce /etc/apparmor.d/usr.sbin.nginx
aa-complain /etc/apparmor.d/usr.sbin.nginx
```

## RBAC — Role-Based Access Control

```
# Components
# ┌────────┐    ┌───────┐    ┌─────────────┐
# │ Users  │───>│ Roles │───>│ Permissions │
# └────────┘    └───────┘    └─────────────┘
#                   │
#               ┌───────────┐
#               │ Sessions  │  (user activates roles per session)
#               └───────────┘

# RBAC Levels (NIST RBAC Standard)
# RBAC0 — Core: users, roles, permissions, sessions
# RBAC1 — Hierarchical: role inheritance (Senior Admin > Admin > Viewer)
# RBAC2 — Constrained: static/dynamic SoD, cardinality constraints
# RBAC3 — Symmetric: RBAC1 + RBAC2 combined

# Role Hierarchy Example
#        CTO
#       /   \
#    DevMgr  SecMgr
#    /    \      \
# Dev   QA    Analyst
#    \  /
#   Viewer

# Constraints
# Static Separation of Duty (SoD)
#   - User cannot be assigned BOTH "Developer" AND "Approver"
#   - Enforced at role assignment time
# Dynamic Separation of Duty
#   - User cannot ACTIVATE conflicting roles in same session
#   - May hold both roles but not use simultaneously
# Cardinality
#   - Max N users per role (e.g., only 2 users can be "Root Admin")
#   - Max N roles per user

# Database RBAC Example (PostgreSQL)
CREATE ROLE readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO readonly;
GRANT readonly TO analyst_user;

# AWS IAM RBAC Pattern
# Role: "S3ReadOnly"
# Policy: { "Effect": "Allow", "Action": "s3:Get*", "Resource": "*" }
# User: analyst → assumes role S3ReadOnly
```

## ABAC — Attribute-Based Access Control

```
# Decision based on attributes of:
#   Subject:     role, department, clearance, location, device
#   Object:      classification, owner, type, sensitivity
#   Environment: time of day, threat level, network zone
#   Action:      read, write, execute, delete

# Policy Example (pseudocode)
# IF subject.role == "doctor"
# AND object.type == "medical_record"
# AND object.patient.hospital == subject.hospital
# AND environment.time IN working_hours
# AND subject.device.compliant == true
# THEN allow READ

# XACML Architecture
# ┌─────────┐    ┌─────────┐    ┌─────────┐
# │  PEP    │───>│  PDP    │<───│  PIP    │
# │ Policy  │    │ Policy  │    │ Policy  │
# │ Enforce │    │ Decision│    │  Info   │
# │ Point   │    │ Point   │    │ Point   │
# └─────────┘    └────┬────┘    └─────────┘
#                     │
#                ┌────┴────┐
#                │  PAP    │
#                │ Policy  │
#                │ Admin   │
#                │ Point   │
#                └─────────┘

# PEP — Intercepts requests, asks PDP, enforces decision
# PDP — Evaluates policies, returns Permit/Deny/NotApplicable/Indeterminate
# PIP — Retrieves attribute values from external sources (LDAP, DB, API)
# PAP — Where admins author and manage policies

# XACML Policy (simplified)
# <Policy PolicyId="medical-record-access">
#   <Target>
#     <Resource>medical_record</Resource>
#   </Target>
#   <Rule Effect="Permit">
#     <Condition>
#       SubjectAttribute("role") == "doctor" AND
#       ResourceAttribute("hospital") == SubjectAttribute("hospital")
#     </Condition>
#   </Rule>
# </Policy>

# Combining Algorithms
# deny-overrides    — any Deny wins
# permit-overrides  — any Permit wins
# first-applicable  — first matching rule wins
# only-one-applicable — exactly one policy must match
```

## ReBAC — Relationship-Based Access Control

```
# Access based on relationships between objects and subjects
# Inspired by Google Zanzibar (used by Google Drive, YouTube, etc.)

# Relationship Tuples
# <object>#<relation>@<subject>
# document:readme#owner@user:alice
# document:readme#viewer@group:engineering#member
# folder:root#parent@document:readme

# Evaluation: "Can user:bob view document:readme?"
# 1. Check: document:readme#viewer@user:bob? — No direct tuple
# 2. Check: document:readme#viewer@group:*#member? — Is bob in any viewer group?
# 3. Check: document:readme#owner@user:bob? — Owners can view (implied)
# 4. Check: folder:root#viewer@user:bob? — Parent folder grants access?

# Open Source Implementations
# OpenFGA (CNCF)     — Zanzibar-inspired, K8s-native
# SpiceDB             — Zanzicar-inspired, gRPC API
# Ory Keto            — Go implementation
# Warrant              — API-first ReBAC

# OpenFGA Model Example (DSL)
# model
#   schema 1.1
# type user
# type document
#   relations
#     define owner: [user]
#     define editor: [user] or owner
#     define viewer: [user] or editor
```

## Rule-Based Access Control

```
# Decisions based on predefined rules (not identity)
# Common in firewalls, routers, WAFs

# Firewall rules (iptables)
iptables -A INPUT -s 10.0.0.0/8 -p tcp --dport 443 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j DROP

# Time-based rules (temporal access control)
# Allow access only during business hours
iptables -A INPUT -p tcp --dport 22 \
  -m time --timestart 09:00 --timestop 17:00 \
  --weekdays Mon,Tue,Wed,Thu,Fri -j ACCEPT

# Context-based access
# Combine location + device + behavior
# - Deny access from unknown geolocations
# - Require MFA from new devices
# - Block if risk score > threshold (risk-adaptive)
```

## Risk-Adaptive Access Control

```
# Dynamic access decisions based on real-time risk scoring

# Risk Factors
# ┌──────────────────────────────────────┐
# │ Factor              │ Weight  │ Score│
# ├──────────────────────────────────────┤
# │ Unknown device      │ High    │ +30  │
# │ Foreign IP          │ High    │ +25  │
# │ Off-hours access    │ Medium  │ +15  │
# │ Sensitive resource  │ Medium  │ +20  │
# │ Failed auth history │ High    │ +25  │
# │ Impossible travel   │ Critical│ +40  │
# └──────────────────────────────────────┘

# Risk Thresholds
# 0-20:   Allow (standard access)
# 21-50:  Step-up authentication (MFA)
# 51-75:  Allow read-only, deny write
# 76-100: Deny + alert SOC
```

## Capability-Based vs ACL-Based

```
# ACL-Based (identity-centric)
# - Permission stored WITH the object
# - "Who can access this file?"
# - Example: Unix file permissions, Windows NTFS
# - Revocation: easy (modify the ACL)
# - Delegation: difficult (must modify ACL)

# Capability-Based (token-centric)
# - Permission stored WITH the subject (as a token/key)
# - "What can this token access?"
# - Example: file descriptors, OAuth tokens, object capabilities
# - Revocation: difficult (must invalidate all copies of token)
# - Delegation: easy (pass the token)

# Comparison
# Feature        ACL              Capability
# ───────        ───              ──────────
# Stored at      Object           Subject
# Question       Who can access?  What can I access?
# Ambient auth   Yes              No
# Confused       Vulnerable       Resistant
#   deputy
# Delegation     Hard             Easy
# Revocation     Easy             Hard
# POLA           Weak             Strong
```

## Implementation Patterns

```
# Middleware pattern (Go/Python/Node)
# 1. Authenticate → identify the subject
# 2. Extract attributes (roles, groups, claims)
# 3. Load policy
# 4. Evaluate: policy(subject, action, object) → permit/deny
# 5. Enforce decision

# Open Policy Agent (OPA) — General-purpose policy engine
# Policy (Rego language)
# package authz
# default allow = false
# allow {
#     input.method == "GET"
#     input.user.role == "viewer"
# }

# Casbin — Multi-model authorization library
# Supports: ACL, RBAC, ABAC, RESTful
# Model file defines: request, policy, matchers, effect
# [request_definition]
# r = sub, obj, act
# [policy_definition]
# p = sub, obj, act
# [matchers]
# m = r.sub == p.sub && r.obj == p.obj && r.act == p.act
```

## See Also

- pam
- selinux
- identity-management
- oauth
- zero-trust
- acl

## References

- NIST RBAC Standard: SP 800-207
- NIST ABAC Guide: SP 800-162
- XACML 3.0: OASIS Standard
- Google Zanzibar Paper: "Zanzibar: Google's Consistent, Global Authorization System" (2019)
- NIST SP 800-53: AC Family (Access Control)
