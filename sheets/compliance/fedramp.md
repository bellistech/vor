# FedRAMP (Federal Risk and Authorization Management Program)

FedRAMP is a US government-wide program that provides a standardized approach to security assessment, authorization, and continuous monitoring for cloud products and services used by federal agencies, built on NIST 800-53 controls with three authorization levels corresponding to data sensitivity.

## Authorization Levels
### Impact Level Comparison
```
                  Low               Moderate            High
─────────────────────────────────────────────────────────────────
Controls          ~130              ~325                ~420
Data sensitivity  Public/non-       CUI, PII, PHI,      National security,
                  sensitive         financial            law enforcement,
                                                        life safety
Agencies          Few (low-risk     Most civilian        DoD, IC, law
                  use cases)        agencies             enforcement
Examples          Public websites,  Email, HR systems,   C2 systems,
                  blogs             financial apps       intelligence tools
Assessment cost   $150K-$250K       $500K-$1.5M         $1.5M-$3M+
Timeline          3-6 months        9-18 months          12-24 months
```

## Authorization Paths
### JAB vs Agency Authorization
```
JAB (Joint Authorization Board) Path:
  1. FedRAMP Connect — Apply and get prioritized
  2. Readiness Assessment — 3PAO validates readiness
  3. Full Security Assessment — 3PAO tests all controls
  4. JAB Review — P-ATO issued by JAB (DoD, DHS, GSA)
  5. Agency adoption — Agencies leverage P-ATO
  Timeline: 12-18 months
  Benefit: Broad reuse across agencies

Agency Authorization Path:
  1. Engage sponsoring agency directly
  2. Agency reviews SSP and documentation
  3. 3PAO conducts assessment
  4. Agency issues ATO
  5. Upload package to FedRAMP Marketplace
  Timeline: 6-12 months
  Benefit: Faster, agency-specific scope
```

## System Security Plan (SSP)
### SSP Template Structure
```bash
# FedRAMP SSP structure (based on Rev 5 templates)
mkdir -p fedramp_package/{ssp,attachments,sar,sap,poam}

cat <<'EOF' > fedramp_package/ssp/ssp_outline.md
# System Security Plan (SSP)

## 1. Information System Name and Identifier
## 2. Information System Categorization (FIPS 199)
## 3. Information System Owner
## 4. Authorizing Official
## 5. System Description
##    5.1 System Function/Purpose
##    5.2 Information System Components
##    5.3 Network Architecture (Diagram required)
##    5.4 Data Flow (Diagram required)
## 6. System Environment and Inventory
## 7. Information Types (NIST 800-60)
## 8. Minimum Security Controls (per baseline)
## 9. Control Implementation Summary
##    - For each control family (AC, AU, CM, etc.):
##      - Control description
##      - Implementation status
##      - Responsible role
##      - Implementation details
## 10. Interconnections (ISAs/MOUs)
## 11. Laws, Regulations, Standards
## 12. Attachments
##     A. Policies and Procedures
##     B. User Guide
##     C. Digital Identity Worksheet
##     D. Privacy Impact Assessment
##     E. Rules of Behavior
##     F. Information System Contingency Plan
##     G. Configuration Management Plan
##     H. Incident Response Plan
##     I. CIS/STIG Hardening Evidence
EOF
```

### Control Implementation (Moderate Baseline)
```bash
# AC-2: Account Management implementation
cat <<'EOF' > fedramp_package/ssp/ac-2_implementation.md
## AC-2 Account Management

**Implementation Status:** Implemented

**Responsible Role:** System Administrator, IAM Team

**Implementation Details:**
- User accounts provisioned via SCIM from agency IdP (AC-2a)
- Account types: privileged, non-privileged, service, emergency (AC-2a)
- Conditions for group membership defined in RBAC policy (AC-2c)
- Accounts reviewed quarterly by IAM team lead (AC-2j)
- Accounts disabled within 24 hours of personnel action (AC-2i)
- Automated notification to ISSO on account events (AC-2(4))
- Service accounts use certificate-based auth, no passwords (AC-2(11))

**Evidence:**
- IdP SCIM integration logs
- Quarterly access review spreadsheets
- Automated offboarding workflow screenshots
- Account activity audit reports
EOF
```

## 3PAO Assessment
### Assessment Process
```
Phase 1: Planning (SAP Development)
  - Scope definition and boundary validation
  - Test case development (per control)
  - Rules of engagement
  - Penetration testing scope

Phase 2: Assessment Execution
  - Document review (SSP, policies, procedures)
  - Interview personnel (system owners, admins, users)
  - Technical testing (vulnerability scans, pen tests)
  - Control testing (per assessment procedures)

Phase 3: Reporting (SAR Development)
  - Security Assessment Report (SAR)
  - Risk Exposure Table
  - Vulnerability scan results
  - Penetration test report
  - Recommendations
```

### Vulnerability Scanning Requirements
```bash
# Monthly vulnerability scanning (FedRAMP requirement)
# High/Critical: 30-day remediation
# Moderate: 90-day remediation
# Low: 180-day remediation

# Nessus scan configuration for FedRAMP
cat <<'EOF' > fedramp_scan_policy.json
{
  "name": "FedRAMP-Monthly-Scan",
  "description": "FedRAMP Moderate baseline vulnerability scan",
  "settings": {
    "scan_type": "authenticated",
    "port_range": "1-65535",
    "severity_threshold": "informational",
    "compliance_checks": [
      "CIS_Benchmark",
      "DISA_STIG"
    ],
    "credentials": {
      "ssh": { "auth_method": "certificate" },
      "windows": { "auth_method": "kerberos" }
    },
    "schedule": {
      "frequency": "monthly",
      "day": 1,
      "notification": ["isso@agency.gov"]
    }
  }
}
EOF

# Track scan results over time
cat <<'EOF' > scan_tracking.sql
CREATE TABLE vulnerability_scans (
  scan_id       UUID PRIMARY KEY,
  scan_date     DATE NOT NULL,
  total_hosts   INT,
  critical      INT DEFAULT 0,
  high          INT DEFAULT 0,
  moderate      INT DEFAULT 0,
  low           INT DEFAULT 0,
  info          INT DEFAULT 0,
  false_pos     INT DEFAULT 0,
  remediated    INT DEFAULT 0
);

-- Monthly trend query
SELECT
  scan_date,
  critical + high AS urgent,
  moderate,
  low,
  ROUND(remediated * 100.0 / NULLIF(critical + high + moderate + low, 0), 1)
    AS remediation_rate
FROM vulnerability_scans
ORDER BY scan_date DESC
LIMIT 12;
EOF
```

## POA&M Management
### Plan of Action and Milestones
```bash
# POA&M tracking spreadsheet structure
cat <<'EOF' > fedramp_package/poam/poam_template.csv
POAM_ID,Weakness,Control,Source,Severity,Status,Scheduled_Completion,Milestone_Changes,Vendor_Dependency,Risk_Adjustment,Comments
POAM-001,"Missing disk encryption on backup server",SC-28,Scan,High,Open,2024-06-30,"Q2: Procure HSM; Q3: Deploy","AWS CloudHSM",No,"Compensating control: encrypted VPN tunnel"
POAM-002,"Stale service accounts not disabled",AC-2,3PAO Finding,Moderate,In Progress,2024-04-15,"Automated cleanup script deployed","None",No,"85% complete"
POAM-003,"Incomplete audit log forwarding",AU-6,ConMon,Moderate,Open,2024-05-30,"Pending SIEM capacity upgrade","Splunk",Yes,"Interim: manual weekly review"
EOF

# POA&M age tracking
cat <<'SCRIPT' > check_poam_age.sh
#!/bin/bash
# Flag overdue POA&M items
while IFS=',' read -r id weakness control source severity status due rest; do
  if [[ "$status" != "Closed" && "$status" != "POAM_ID" ]]; then
    due_epoch=$(date -j -f "%Y-%m-%d" "$due" "+%s" 2>/dev/null)
    now_epoch=$(date "+%s")
    if [[ $due_epoch -lt $now_epoch ]]; then
      days_overdue=$(( (now_epoch - due_epoch) / 86400 ))
      echo "OVERDUE: $id ($severity) - $days_overdue days - $weakness"
    fi
  fi
done < fedramp_package/poam/poam_template.csv
SCRIPT
chmod +x check_poam_age.sh
```

## Continuous Monitoring
### ConMon Requirements
```
Monthly:
  - Vulnerability scanning (all components)
  - POA&M updates and status reporting
  - Unique vulnerability count and remediation rates

Quarterly:
  - Privileged access reviews
  - Security configuration compliance checks (STIG/CIS)

Annually:
  - Penetration testing (external and internal)
  - Contingency plan testing
  - Incident response plan testing
  - Security awareness training
  - Annual assessment (subset of controls)

Ongoing:
  - Significant change requests (within 30 days of change)
  - Incident reporting (US-CERT timelines)
  - Configuration change documentation
```

### Automated ConMon Dashboard
```bash
# ConMon metrics collection script
cat <<'EOF' > conmon_metrics.sh
#!/bin/bash
DATE=$(date +%Y-%m-%d)

echo "=== FedRAMP ConMon Report: $DATE ==="

# Vulnerability counts from scan results
echo "## Vulnerability Summary"
echo "Critical: $(grep -c 'Critical' scan_results.csv)"
echo "High: $(grep -c 'High' scan_results.csv)"
echo "Moderate: $(grep -c 'Moderate' scan_results.csv)"

# SLA compliance
echo "## Remediation SLA Compliance"
echo "Critical/High (30-day SLA): $(
  awk -F',' '$5=="Critical" || $5=="High" {
    if ($8 <= 30) pass++; total++
  } END { printf "%.1f%%\n", pass/total*100 }' remediation_log.csv
)"

# POA&M status
echo "## POA&M Status"
echo "Open: $(grep -c 'Open' poam_template.csv)"
echo "In Progress: $(grep -c 'In Progress' poam_template.csv)"
echo "Overdue: $(./check_poam_age.sh | wc -l)"
EOF
```

## Rev 5 Alignment
### Key Changes in FedRAMP Rev 5
```
- Aligned to NIST 800-53 Rev 5 (from Rev 4)
- New control family: SR (Supply Chain Risk Management)
- New control family: PT (PII Processing and Transparency)
- Enhanced privacy controls throughout
- Consolidation of program/management controls under PM
- Updated automation requirements for ConMon
- New OSCAL-based documentation format support
- Updated penetration testing guidance
- Strengthened incident response timelines
```

## Tips
- Engage a 3PAO early in the process for a readiness assessment before committing to a full authorization
- Use the FedRAMP Marketplace to study comparable CSP authorizations for scoping guidance
- Automate evidence collection for continuous monitoring from day one; manual processes do not scale
- Maintain a living SSP; treat it as code and version control it alongside your infrastructure
- Map your existing SOC 2 or ISO 27001 controls to FedRAMP to identify gaps rather than starting from zero
- Budget for 3PAO costs, which typically range from $200K to $500K for moderate baseline assessments
- Track POA&M items rigorously; overdue items are the number one finding in annual assessments
- Consider the Agency path for faster authorization if you have a willing sponsoring agency
- Use FedRAMP OSCAL templates for machine-readable SSP generation and validation
- Keep your boundary diagram current; inaccurate boundaries are a common cause of authorization delays
- Prepare for Rev 5 migration by auditing your current controls against the new supply chain (SR) family

## See Also
- nist, soc2, gdpr, cis-benchmarks, cloud-security

## References
- [FedRAMP Official Website](https://www.fedramp.gov/)
- [FedRAMP Marketplace](https://marketplace.fedramp.gov/)
- [FedRAMP Rev 5 Baselines](https://www.fedramp.gov/baselines/)
- [FedRAMP Authorization Playbook](https://www.fedramp.gov/assets/resources/documents/CSP_Authorization_Playbook_Getting_Started_with_FedRAMP.pdf)
- [NIST SP 800-53 Rev 5](https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final)
