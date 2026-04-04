# SOC 2 (Service Organization Control 2)

SOC 2 is an auditing framework developed by AICPA that evaluates a service organization's controls relevant to security, availability, processing integrity, confidentiality, and privacy, commonly required by enterprise customers evaluating SaaS and cloud service providers.

## Trust Service Criteria (TSC)
### The Five Pillars
```
Security (Common Criteria)     — Required for ALL SOC 2 reports
Availability                   — Uptime, disaster recovery, failover
Processing Integrity           — Accuracy, completeness, timeliness
Confidentiality                — Encryption, access controls, NDA enforcement
Privacy                        — PII handling, consent, data subject rights
```

### Common Criteria Categories (CC1-CC9)
```
CC1  — Control Environment (tone at the top, governance)
CC2  — Communication and Information (policies, awareness)
CC3  — Risk Assessment (threat identification, risk appetite)
CC4  — Monitoring Activities (continuous monitoring, deficiency tracking)
CC5  — Control Activities (logical access, change mgmt)
CC6  — Logical and Physical Access Controls (authentication, MFA)
CC7  — System Operations (incident detection, response)
CC8  — Change Management (SDLC, code review, deployment)
CC9  — Risk Mitigation (vendor management, business continuity)
```

## Type I vs Type II
### Comparison
```
                  Type I                    Type II
─────────────────────────────────────────────────────────
Scope             Design of controls        Design + Operating effectiveness
Time window       Point-in-time (snapshot)  Period of time (3-12 months)
Typical duration  1-3 months prep           6-12 months observation
Cost              $20K-$50K                 $50K-$150K+
Customer trust    Lower (no operational     Higher (proven controls
                  evidence)                 over time)
First-time orgs   Start here                Graduate to Type II
```

### Audit Readiness Assessment
```bash
# Sample readiness checklist (scored 0-5 per control)
cat <<'EOF' > soc2_readiness.csv
Control,Category,Score,Gap,Remediation
"Access Reviews",CC6,3,"Quarterly not monthly","Automate with IdP"
"Change Mgmt",CC8,4,"Missing rollback docs","Add runbook template"
"Incident Response",CC7,2,"No tabletop exercises","Schedule quarterly drills"
"Vendor Risk",CC9,1,"No vendor inventory","Deploy vendor risk platform"
"Encryption at Rest",CC6,5,"None","N/A"
"MFA Enforcement",CC6,4,"Service accounts exempt","Implement cert-based auth"
"Log Retention",CC7,3,"30 days only","Extend to 365 days"
"BCP/DR Testing",CC3,2,"Never tested","Annual DR failover test"
EOF
```

## Controls Mapping
### Access Management (CC6)
```bash
# Enforce MFA across identity provider (Okta example)
curl -X PUT "https://${OKTA_DOMAIN}/api/v1/policies/${POLICY_ID}" \
  -H "Authorization: SSWS ${OKTA_API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "SOC2-MFA-Policy",
    "status": "ACTIVE",
    "conditions": {
      "people": { "groups": { "include": ["EVERYONE"] } }
    },
    "settings": {
      "factors": {
        "okta_otp": { "enroll": { "self": "REQUIRED" } },
        "okta_push": { "enroll": { "self": "REQUIRED" } }
      }
    }
  }'

# Quarterly access review query (AWS IAM)
aws iam generate-credential-report
aws iam get-credential-report --output text \
  --query 'Content' | base64 -d > iam_report.csv

# Find users with console access but no MFA
awk -F',' '$4=="true" && $8=="false" {print $1}' iam_report.csv
```

### Change Management (CC8)
```bash
# Enforce branch protection (GitHub API)
curl -X PUT \
  "https://api.github.com/repos/${ORG}/${REPO}/branches/main/protection" \
  -H "Authorization: token ${GITHUB_TOKEN}" \
  -d '{
    "required_status_checks": {
      "strict": true,
      "contexts": ["ci/tests", "security/scan"]
    },
    "enforce_admins": true,
    "required_pull_request_reviews": {
      "required_approving_review_count": 2,
      "dismiss_stale_reviews": true
    },
    "restrictions": null
  }'
```

### Monitoring and Logging (CC7)
```bash
# CloudWatch log retention policy (365 days for SOC 2)
aws logs put-retention-policy \
  --log-group-name "/app/production" \
  --retention-in-days 365

# Alert on unauthorized access attempts
aws cloudwatch put-metric-alarm \
  --alarm-name "SOC2-UnauthorizedAccess" \
  --metric-name "UnauthorizedAccessCount" \
  --namespace "Security" \
  --statistic Sum \
  --period 300 \
  --threshold 10 \
  --comparison-operator GreaterThanThreshold \
  --alarm-actions "${SNS_TOPIC_ARN}"
```

## Evidence Collection
### Automated Evidence Gathering
```bash
# Collect evidence artifacts for auditor
mkdir -p soc2_evidence/{access,change,incident,vendor}

# Access reviews
aws iam generate-credential-report
aws iam get-credential-report --output text \
  --query 'Content' | base64 -d \
  > soc2_evidence/access/iam_credential_report_$(date +%Y%m%d).csv

# Change management logs
git log --since="6 months ago" --format="%H|%an|%ae|%aI|%s" \
  > soc2_evidence/change/git_commits.csv

# Pull request review evidence
curl -s "https://api.github.com/repos/${ORG}/${REPO}/pulls?state=closed&per_page=100" \
  -H "Authorization: token ${GITHUB_TOKEN}" \
  | jq '[.[] | {number, title, user: .user.login,
    merged_at, review_comments}]' \
  > soc2_evidence/change/pr_reviews.json

# Incident response logs
aws support describe-trusted-advisor-checks \
  --language en \
  > soc2_evidence/incident/advisor_checks.json
```

## Audit Preparation Timeline
### 12-Month Plan
```
Month 1-2:   Gap assessment, scope definition, select auditor
Month 3-4:   Policy creation/updates, control implementation
Month 5-6:   Tool deployment (SIEM, MDM, vulnerability scanner)
Month 7-8:   Employee training, tabletop exercises
Month 9:     Internal audit / readiness assessment
Month 10-11: Observation period begins (Type II)
Month 12:    Auditor fieldwork, report generation
Ongoing:     Continuous monitoring, quarterly reviews
```

## Continuous Compliance Tools
### Policy-as-Code (Open Policy Agent)
```rego
# soc2_access_policy.rego — Enforce least privilege
package soc2.access

default allow = false

# CC6.1: Restrict access to authorized users
allow {
    input.user.role == "admin"
    input.resource.classification != "restricted"
}

allow {
    input.user.role == "developer"
    input.resource.type == "code_repository"
    input.action in ["read", "write"]
}

# CC6.3: Deny access after termination
deny[msg] {
    input.user.status == "terminated"
    msg := sprintf("Terminated user %s attempted access", [input.user.id])
}
```

### Compliance Dashboard Query (SQL)
```sql
-- Control effectiveness summary
SELECT
    control_id,
    control_name,
    category,
    COUNT(CASE WHEN status = 'pass' THEN 1 END) AS passing,
    COUNT(CASE WHEN status = 'fail' THEN 1 END) AS failing,
    ROUND(
      COUNT(CASE WHEN status = 'pass' THEN 1 END) * 100.0 /
      COUNT(*), 1
    ) AS pass_rate
FROM compliance_checks
WHERE framework = 'SOC2'
  AND check_date >= DATE_SUB(CURDATE(), INTERVAL 90 DAY)
GROUP BY control_id, control_name, category
ORDER BY pass_rate ASC;
```

## Tips
- Start with Type I to establish a baseline, then graduate to Type II within 12 months
- Automate evidence collection from day one to avoid last-minute scrambles during audit windows
- Map your existing controls to TSC criteria before buying any GRC tooling
- Use a shared evidence repository (e.g., Vanta, Drata, or even a structured S3 bucket) accessible to auditors
- Conduct quarterly access reviews, not just annual, to stay ahead of CC6 requirements
- Document your risk assessment process thoroughly; auditors care more about the process than the outcome
- Train all employees on security awareness annually and keep completion certificates as evidence
- Include subservice organizations (hosting providers, SaaS dependencies) in your scope or use a carve-out
- Build SOC 2 controls into your CI/CD pipeline so compliance becomes part of the development workflow
- Maintain an exception log for any control deviations with documented compensating controls
- Schedule a pre-audit readiness assessment 60-90 days before the formal audit begins

## See Also
- nist, fedramp, gdpr, iso27001, hipaa

## References
- [AICPA SOC 2 Overview](https://www.aicpa-cima.com/topic/audit-assurance/audit-and-assurance-greater-than-soc-2)
- [AICPA Trust Services Criteria (2017)](https://us.aicpa.org/content/dam/aicpa/interestareas/frc/assuranceadvisoryservices/downloadabledocuments/trust-services-criteria.pdf)
- [SOC 2 Compliance Handbook — Vanta](https://www.vanta.com/collection/soc-2)
- [Cloud Security Alliance CCM to SOC 2 Mapping](https://cloudsecurityalliance.org/research/cloud-controls-matrix)
