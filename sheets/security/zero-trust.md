# Zero Trust Architecture

Zero trust is a security model that eliminates implicit trust by enforcing continuous verification of every user, device, and network flow, implementing least-privilege access through identity-centric policies, microsegmentation, and the PEP/PDP/PA architecture defined in NIST SP 800-207.

## Core Principles

### Never Trust, Always Verify

```bash
# Zero trust eliminates the trusted network perimeter
# Every request must be authenticated, authorized, and encrypted
# regardless of source location (internal or external)

# Traditional perimeter model:
#   [Internet] -> [Firewall] -> [Trusted Internal Network]
#                                  (implicit trust)

# Zero trust model:
#   [Any Network] -> [PEP] -> [PDP] -> [Resource]
#                    verify    decide    grant/deny
#                    identity  policy    per-request

# NIST SP 800-207 tenets:
# 1. All data sources and computing services are resources
# 2. All communication is secured regardless of location
# 3. Access is granted on a per-session basis
# 4. Access is determined by dynamic policy
# 5. Enterprise monitors and measures integrity of all assets
# 6. Authentication and authorization are strictly enforced
# 7. Enterprise collects information for improving security posture
```

## PEP/PDP/PA Architecture

### NIST 800-207 Components

```bash
# Policy Decision Point (PDP) / Policy Engine (PE)
# - Evaluates access requests against policy
# - Considers identity, device posture, context, threat intel
# - Returns allow/deny/conditional decisions

# Policy Administrator (PA)
# - Executes PDP decisions
# - Configures data plane (PEP) to allow/deny traffic
# - Manages session tokens and credentials

# Policy Enforcement Point (PEP)
# - Inline data plane component
# - Enables/disables/isolates connections
# - Enforces PDP decisions at the network level

# Request flow:
# Subject -> PEP -> PA -> PDP -> Policy Store
#                              -> Trust Algorithm
#                              -> Threat Intelligence
#                              -> Activity Logs
```

### Open Policy Agent (OPA) as PDP

```bash
# Install OPA
brew install opa

# Define access policy (policy.rego)
cat <<'REGO' > policy.rego
package authz

default allow := false

allow if {
    input.user.authenticated == true
    input.user.department == input.resource.department
    input.device.compliant == true
    input.risk_score < 70
}

deny_reasons contains reason if {
    not input.user.authenticated
    reason := "user not authenticated"
}

deny_reasons contains reason if {
    not input.device.compliant
    reason := "device not compliant"
}
REGO

# Evaluate policy
opa eval -i input.json -d policy.rego "data.authz.allow"

# Run OPA as server (PDP endpoint)
opa run --server --addr :8181 policy.rego

# Query PDP
curl -X POST http://localhost:8181/v1/data/authz/allow \
  -H "Content-Type: application/json" \
  -d '{"input": {"user": {"authenticated": true, "department": "eng"}, "device": {"compliant": true}, "risk_score": 30, "resource": {"department": "eng"}}}'
```

## Identity-Centric Access

### Identity Provider Integration

```bash
# OIDC-based identity verification (example with Keycloak)
# 1. User authenticates with IdP
# 2. IdP issues JWT with claims
# 3. PEP validates JWT on every request
# 4. PDP evaluates claims against policy

# Validate JWT token (using step-cli)
step crypto jwt verify \
  --iss https://idp.example.com/realms/corp \
  --aud myapp \
  --alg RS256 \
  --key /path/to/public-key.pem \
  < token.jwt

# Extract identity claims
step crypto jwt inspect --insecure < token.jwt | jq '.payload'

# SPIFFE/SPIRE for workload identity
# Install SPIRE server
curl -fsSL https://github.com/spiffe/spire/releases/download/v1.9.0/spire-1.9.0-linux-amd64-musl.tar.gz \
  | tar xz

# Register workload identity
spire-server entry create \
  -spiffeID spiffe://example.com/myapp \
  -parentID spiffe://example.com/node1 \
  -selector k8s:pod-label:app:myapp

# Fetch SVID (workload identity document)
spire-agent api fetch x509 -write /tmp/
```

## Microsegmentation

### Network Policy (Kubernetes)

```yaml
# Deny all ingress by default (zero trust baseline)
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-all
  namespace: production
spec:
  podSelector: {}
  policyTypes:
    - Ingress
    - Egress
---
# Allow specific service-to-service communication
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-frontend-to-api
  namespace: production
spec:
  podSelector:
    matchLabels:
      app: api
  policyTypes:
    - Ingress
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app: frontend
      ports:
        - port: 8080
          protocol: TCP
```

### Cilium L7 Network Policy

```yaml
# L7-aware microsegmentation with Cilium
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: api-l7-policy
spec:
  endpointSelector:
    matchLabels:
      app: api
  ingress:
    - fromEndpoints:
        - matchLabels:
            app: frontend
      toPorts:
        - ports:
            - port: "8080"
              protocol: TCP
          rules:
            http:
              - method: GET
                path: "/api/v1/public/.*"
              - method: POST
                path: "/api/v1/data"
                headers:
                  - 'X-Auth-Token: [a-zA-Z0-9]+'
```

## Device Posture

### Device Trust Assessment

```bash
# Device posture signals to evaluate:
# - OS patch level and update status
# - Disk encryption enabled (FileVault, BitLocker, LUKS)
# - Endpoint protection running and up-to-date
# - Screen lock configured
# - Firewall enabled
# - Not jailbroken/rooted
# - Certificate validity

# Check macOS device posture (example script)
# Disk encryption
fdesetup status  # FileVault is On/Off

# Firewall status
/usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate

# OS version
sw_vers -productVersion

# MDM enrollment check
profiles status -type enrollment

# Combine into posture score for PDP
# Score = sum of compliant checks / total checks
# Feed score into policy engine for access decisions
```

## Continuous Verification

### Runtime Trust Evaluation

```bash
# Zero trust requires continuous (not one-time) verification
# Re-evaluate trust on every request or at short intervals

# Signals for continuous evaluation:
# - Time since last authentication
# - Geographic location changes
# - Behavior anomalies (unusual access patterns)
# - Threat intelligence updates
# - Device posture changes
# - Network context changes

# Example: session risk scoring
# risk_score = f(time_since_auth, geo_anomaly, behavior_score, device_posture)
# if risk_score > threshold: step-up auth or deny

# Implement with service mesh (Istio/Envoy)
# - mTLS between all services
# - JWT validation on every request
# - Rate limiting per identity
# - Authorization policy per endpoint
```

### BeyondCorp Implementation

```bash
# Google BeyondCorp principles:
# 1. Access depends on device and user credentials
# 2. Access is granted on a per-request basis
# 3. All access to resources is fully authenticated, authorized, encrypted
# 4. Access determined by dynamic policy
# 5. Device inventory informs access decisions
# 6. Trust is not binary but a spectrum

# BeyondCorp proxy pattern:
#   User -> Identity-Aware Proxy -> Access Policy Engine -> Resource
#           (authenticates)         (authorizes)           (serves)

# Open source BeyondCorp alternatives:
# - Pomerium (identity-aware proxy)
# - Teleport (zero trust access plane)
# - Boundary (HashiCorp identity-based access)

# Pomerium configuration example
# pomerium-config.yaml
# routes:
#   - from: https://app.example.com
#     to: http://internal-app:8080
#     policy:
#       - allow:
#           and:
#             - email:
#                 is: user@example.com
#             - device:
#                 is: approved
```

## Tips

- Start zero trust with identity: strong authentication (MFA/FIDO2) is the foundation for everything else
- Default-deny all network traffic and explicitly allow only verified service-to-service flows
- Use SPIFFE/SPIRE for workload identity instead of relying on network location for service trust
- Implement microsegmentation incrementally; start with the most sensitive workloads first
- Device posture assessment must be continuous, not just at login time
- Encrypt all traffic with mTLS even inside the "internal" network; zero trust assumes the network is hostile
- Use a dedicated policy engine (OPA, Cedar) as the PDP rather than embedding policy in application code
- Log every access decision (allow and deny) for forensics and continuous improvement
- Step-up authentication for high-risk operations rather than blanket MFA for everything
- Zero trust is a journey: prioritize by risk, implement incrementally, measure continuously
- BeyondCorp-style proxies (Pomerium, Teleport) are the fastest path for web application zero trust

## See Also

- network-defense, pki, oauth, jwt, cilium, wireguard, opa, ids-ips

## References

- [NIST SP 800-207: Zero Trust Architecture](https://csrc.nist.gov/publications/detail/sp/800-207/final)
- [Google BeyondCorp Papers](https://cloud.google.com/beyondcorp)
- [CISA Zero Trust Maturity Model](https://www.cisa.gov/zero-trust-maturity-model)
- [SPIFFE Specification](https://spiffe.io/docs/latest/spiffe-about/overview/)
- [Pomerium - Identity-Aware Proxy](https://www.pomerium.com/docs/)
- [DoD Zero Trust Reference Architecture](https://dodcio.defense.gov/Portals/0/Documents/Library/DoD-ZTRef-Arch.pdf)
