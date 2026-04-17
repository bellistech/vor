# NIST Cybersecurity Framework (CSF 2.0 and 800-53)

NIST provides the foundational cybersecurity frameworks used by US federal agencies and widely adopted by the private sector, encompassing the Cybersecurity Framework (CSF 2.0) with its six core functions, the comprehensive 800-53 control catalog, and the 800-171 requirements for protecting Controlled Unclassified Information.

## CSF 2.0 Core Functions
### The Six Functions
```
GOVERN (GV)    — Establish cybersecurity risk management strategy
                 and supply chain risk management
IDENTIFY (ID)  — Asset management, risk assessment, improvement
PROTECT (PR)   — Access control, awareness, data security, platform security
DETECT (DE)    — Continuous monitoring, adverse event analysis
RESPOND (RS)   — Incident management, analysis, mitigation, reporting
RECOVER (RC)   — Recovery planning, execution, communication
```

### Function-Category Mapping
```
GV.OC  — Organizational Context       GV.RM — Risk Management Strategy
GV.RR  — Roles and Responsibilities   GV.PO — Policy
GV.OV  — Oversight                    GV.SC — Supply Chain Risk Mgmt

ID.AM  — Asset Management             ID.RA — Risk Assessment
ID.IM  — Improvement

PR.AA  — Identity Mgmt & Access Ctrl  PR.AT — Awareness and Training
PR.DS  — Data Security                PR.PS — Platform Security
PR.IR  — Technology Infrastructure Resilience

DE.CM  — Continuous Monitoring         DE.AE — Adverse Event Analysis

RS.MA  — Incident Management          RS.AN — Incident Analysis
RS.CO  — Incident Response Reporting   RS.MI — Incident Mitigation

RC.RP  — Incident Recovery Plan Execution
RC.CO  — Incident Recovery Communication
```

## NIST 800-53 Control Families
### Rev 5 Control Families (20 families)
```
AC  — Access Control (25 controls)         AU — Audit and Accountability (16)
AT  — Awareness and Training (6)           CA — Assessment and Authorization (9)
CM  — Configuration Management (14)        CP — Contingency Planning (13)
IA  — Identification and Authentication (12) IR — Incident Response (10)
MA  — Maintenance (7)                      MP — Media Protection (8)
PE  — Physical and Environmental (23)      PL — Planning (11)
PM  — Program Management (32)              PS — Personnel Security (9)
PT  — PII Processing and Transparency (8)  RA — Risk Assessment (10)
SA  — System and Services Acquisition (23) SC — System and Communications (51)
SI  — System and Information Integrity (23) SR — Supply Chain Risk Mgmt (12)
```

### Control Baselines
```bash
# Query control baselines using OSCAL (JSON format)
# Low baseline (~130 controls), Moderate (~325), High (~420)

# Download NIST 800-53 OSCAL catalog
curl -sL "https://raw.githubusercontent.com/usnistgov/oscal-content/main/nist.gov/SP800-53/rev5/json/NIST_SP-800-53_rev5_catalog.json" \
  -o nist_catalog.json

# Extract controls for a specific family
jq '.catalog.groups[] | select(.id=="ac") | .controls[] | {id, title}' \
  nist_catalog.json

# Count controls per baseline
jq '[.catalog.groups[].controls[] |
  select(.props[]? | select(.name=="label"))] | length' \
  nist_catalog.json
```

### Key Control Implementations
```bash
# AC-2: Account Management
# Automated account provisioning via SCIM
curl -X POST "https://idp.example.com/scim/v2/Users" \
  -H "Authorization: Bearer ${SCIM_TOKEN}" \
  -H "Content-Type: application/scim+json" \
  -d '{
    "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
    "userName": "jdoe@example.com",
    "active": true,
    "emails": [{"value": "jdoe@example.com", "primary": true}],
    "groups": [{"value": "readonly-users"}]
  }'

# AC-6: Least Privilege — AWS IAM policy
cat <<'EOF' > least_privilege_policy.json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "NIST-AC6-LeastPrivilege",
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::project-data",
        "arn:aws:s3:::project-data/*"
      ],
      "Condition": {
        "IpAddress": { "aws:SourceIp": "10.0.0.0/8" }
      }
    }
  ]
}
EOF

aws iam put-user-policy \
  --user-name jdoe \
  --policy-name NIST-AC6-ReadOnly \
  --policy-document file://least_privilege_policy.json

# AU-2/AU-3: Audit Events — CloudTrail configuration
aws cloudtrail create-trail \
  --name nist-audit-trail \
  --s3-bucket-name audit-logs-bucket \
  --is-multi-region-trail \
  --enable-log-file-validation \
  --include-global-service-events

aws cloudtrail start-logging --name nist-audit-trail

# CM-2: Baseline Configuration — Ansible enforcement
cat <<'EOF' > baseline_config.yml
---
- name: NIST CM-2 Baseline Configuration
  hosts: all
  become: yes
  tasks:
    - name: Ensure NTP is configured (CM-2)
      lineinfile:
        path: /etc/chrony.conf
        line: "server time.nist.gov iburst"
    - name: Disable unnecessary services (CM-7)
      systemd:
        name: "{{ item }}"
        enabled: no
        state: stopped
      loop:
        - cups
        - avahi-daemon
        - bluetooth
    - name: Set password complexity (IA-5)
      lineinfile:
        path: /etc/security/pwquality.conf
        regexp: "^{{ item.key }}"
        line: "{{ item.key }} = {{ item.value }}"
      loop:
        - { key: "minlen", value: "14" }
        - { key: "dcredit", value: "-1" }
        - { key: "ucredit", value: "-1" }
        - { key: "ocredit", value: "-1" }
EOF
```

## NIST 800-171 for CUI
### 14 Control Families (110 Requirements)
```
3.1  Access Control (22)           3.2  Awareness & Training (3)
3.3  Audit & Accountability (9)    3.4  Configuration Mgmt (9)
3.5  Identification & Auth (11)    3.6  Incident Response (3)
3.7  Maintenance (6)               3.8  Media Protection (9)
3.9  Personnel Security (2)        3.10 Physical Protection (6)
3.11 Risk Assessment (3)           3.12 Security Assessment (4)
3.13 SC Protection (16)            3.14 SI Integrity (7)
```

### CMMC 2.0 Level Mapping
```
CMMC Level 1 (Foundational) — 17 practices (FCI only)
CMMC Level 2 (Advanced)     — 110 practices = NIST 800-171 (CUI)
CMMC Level 3 (Expert)       — 110 + selected 800-172 enhanced controls
```

## Risk Assessment (RA family)
### NIST Risk Assessment Process
```
Step 1: System Characterization (scope, boundaries, data flows)
Step 2: Threat Identification (STRIDE, MITRE ATT&CK mapping)
Step 3: Vulnerability Identification (CVE scanning, pen testing)
Step 4: Control Analysis (existing controls, planned controls)
Step 5: Likelihood Determination (High/Moderate/Low)
Step 6: Impact Analysis (High/Moderate/Low per CIA triad)
Step 7: Risk Determination (Likelihood x Impact matrix)
Step 8: Control Recommendations (cost-benefit analysis)
Step 9: Documentation (risk register, POA&M)
```

## Continuous Monitoring (ISCM)
### NIST 800-137 Implementation
```bash
# Continuous monitoring dashboard metrics
cat <<'EOF' > iscm_metrics.yaml
vulnerability_management:
  scan_frequency: "weekly"
  critical_patch_sla: "72_hours"
  high_patch_sla: "30_days"
  metric: "percentage_patched_within_sla"
  target: 95

configuration_compliance:
  scan_frequency: "daily"
  baseline: "CIS_Level_2"
  metric: "percentage_compliant_systems"
  target: 98

access_management:
  review_frequency: "quarterly"
  metric: "stale_accounts_percentage"
  target_max: 2

incident_detection:
  mean_time_to_detect: "1_hour"
  mean_time_to_respond: "4_hours"
  false_positive_rate_max: 10
EOF

# Automated SCAP scan with OpenSCAP
oscap xccdf eval \
  --profile xccdf_org.ssgproject.content_profile_stig \
  --results scan_results.xml \
  --report scan_report.html \
  /usr/share/xml/scap/ssg/content/ssg-rhel9-ds.xml
```

## Tips
- Start with CSF 2.0 for strategic alignment, then map down to 800-53 controls for implementation detail
- Use OSCAL (Open Security Controls Assessment Language) for machine-readable compliance documentation
- Select control baselines based on system categorization using FIPS 199 (Low/Moderate/High impact)
- Leverage NIST's free SP 800-53A assessment procedures to build your audit test plans
- Map MITRE ATT&CK techniques to NIST controls for threat-informed defense
- Use SCAP-validated tools (OpenSCAP, Tenable) for automated configuration compliance checking
- Implement continuous monitoring (800-137) rather than point-in-time assessments
- Document control inheritance from cloud providers using their shared responsibility models
- For 800-171 compliance, use the NIST self-assessment handbook (SP 800-171A) to score each requirement
- Review the NIST Privacy Framework alongside CSF for comprehensive data protection coverage
- Apply tailoring guidance to remove controls not applicable to your system's risk profile

## See Also
- soc2, fedramp, gdpr, cis-benchmarks, mitre-attack, dora, eu-ai-act, post-quantum-crypto

## References
- [NIST Cybersecurity Framework 2.0](https://www.nist.gov/cyberframework)
- [NIST SP 800-53 Rev 5](https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final)
- [NIST SP 800-171 Rev 2](https://csrc.nist.gov/publications/detail/sp/800-171/rev-2/final)
- [NIST SP 800-137 ISCM](https://csrc.nist.gov/publications/detail/sp/800-137/final)
- [OSCAL Project](https://pages.nist.gov/OSCAL/)
