# cert-manager

Automated X.509 certificate management for Kubernetes using custom resources and pluggable issuers.

## Installation

```bash
# Install cert-manager with CRDs
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.5/cert-manager.yaml

# Install via Helm
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set crds.enabled=true

# Verify installation
kubectl get pods -n cert-manager
cmctl check api
```

## Issuer and ClusterIssuer

```bash
# Create a self-signed ClusterIssuer
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: selfsigned-issuer
spec:
  selfSigned: {}
EOF

# Create a namespace-scoped Issuer with ACME (Let's Encrypt staging)
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: letsencrypt-staging
  namespace: default
spec:
  acme:
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-staging-key
    solvers:
    - http01:
        ingress:
          class: nginx
EOF

# Let's Encrypt production issuer
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-prod-key
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

## DNS01 Challenge Solvers

```bash
# ClusterIssuer with Route53 DNS01 solver
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-dns
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-dns-key
    solvers:
    - dns01:
        route53:
          region: us-east-1
          hostedZoneID: Z0123456789ABCDEF
      selector:
        dnsZones:
        - "example.com"
EOF

# CloudDNS solver
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-clouddns
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-clouddns-key
    solvers:
    - dns01:
        cloudDNS:
          project: my-gcp-project
          serviceAccountSecretRef:
            name: clouddns-sa
            key: key.json
EOF
```

## Certificate Resources

```bash
# Request a certificate
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: example-tls
  namespace: default
spec:
  secretName: example-tls-secret
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  commonName: example.com
  dnsNames:
  - example.com
  - www.example.com
  duration: 2160h    # 90 days
  renewBefore: 720h  # 30 days before expiry
EOF

# Wildcard certificate (requires DNS01)
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: wildcard-tls
  namespace: default
spec:
  secretName: wildcard-tls-secret
  issuerRef:
    name: letsencrypt-dns
    kind: ClusterIssuer
  dnsNames:
  - "*.example.com"
  - example.com
EOF

# Check certificate status
kubectl get certificate -A
kubectl describe certificate example-tls
kubectl get certificaterequest -A
```

## CA Issuer and Vault Integration

```bash
# Create a CA issuer from an existing CA
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: ca-issuer
spec:
  ca:
    secretName: ca-key-pair
EOF

# Generate the CA key pair secret
openssl genrsa -out ca.key 4096
openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 \
  -subj "/CN=My Internal CA" -out ca.crt
kubectl create secret tls ca-key-pair \
  --cert=ca.crt --key=ca.key -n cert-manager

# HashiCorp Vault issuer
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: vault-issuer
spec:
  vault:
    server: https://vault.example.com
    path: pki_int/sign/example-dot-com
    auth:
      kubernetes:
        role: cert-manager
        mountPath: /v1/auth/kubernetes
        serviceAccountRef:
          name: cert-manager
EOF
```

## Ingress Annotations

```bash
# Annotate an Ingress to auto-provision certificates
cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example-ingress
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    cert-manager.io/common-name: "example.com"
    cert-manager.io/duration: "2160h"
    cert-manager.io/renew-before: "720h"
spec:
  tls:
  - hosts:
    - example.com
    secretName: example-tls-auto
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: my-service
            port:
              number: 80
EOF
```

## Troubleshooting and Lifecycle

```bash
# Inspect the full certificate chain
kubectl get secret example-tls-secret -o jsonpath='{.data.tls\.crt}' | \
  base64 -d | openssl x509 -text -noout

# Check certificate readiness
kubectl get certificate -o wide
kubectl describe certificaterequest <name>

# Check ACME orders and challenges
kubectl get orders -A
kubectl get challenges -A
kubectl describe challenge <name>

# Force certificate renewal
cmctl renew example-tls -n default

# View cert-manager controller logs
kubectl logs -n cert-manager deploy/cert-manager -f

# Check issuer status
kubectl get clusterissuer -o wide
kubectl describe issuer letsencrypt-staging
```

## trust-manager

```bash
# Install trust-manager
helm install trust-manager jetstack/trust-manager \
  --namespace cert-manager \
  --wait

# Create a Bundle to distribute CA certificates
cat <<EOF | kubectl apply -f -
apiVersion: trust.cert-manager.io/v1alpha1
kind: Bundle
metadata:
  name: public-cas
spec:
  sources:
  - useDefaultCAs: true
  - secret:
      name: "ca-key-pair"
      key: "tls.crt"
  target:
    configMap:
      key: "ca-certificates.crt"
    namespaceSelector:
      matchLabels:
        trust-bundle: "true"
EOF
```

## Tips

- Always start with the Let's Encrypt staging server to avoid rate limits during testing
- Use `ClusterIssuer` for org-wide issuers and `Issuer` for namespace-isolated teams
- DNS01 solvers are required for wildcard certificates; HTTP01 cannot validate them
- Set `renewBefore` to at least 30 days so failures have time to be detected and fixed
- The `cmctl` CLI tool is invaluable for checking API readiness and forcing renewals
- Monitor `certificate_expiration_timestamp_seconds` Prometheus metric for expiry alerts
- Separate staging and production issuers to avoid polluting prod with test certificates
- Use trust-manager to distribute internal CA bundles across namespaces automatically
- Check `Orders` and `Challenges` resources when ACME issuance stalls
- Keep cert-manager updated; each minor release often fixes critical ACME edge cases
- Use `revisionHistoryLimit` on Certificate resources to limit stored CertificateRequests

## See Also

- external-secrets
- gateway-api
- kyverno

## References

- [cert-manager Documentation](https://cert-manager.io/docs/)
- [cert-manager GitHub Repository](https://github.com/cert-manager/cert-manager)
- [Let's Encrypt ACME Protocol](https://letsencrypt.org/docs/)
- [trust-manager Documentation](https://cert-manager.io/docs/trust/trust-manager/)
- [cmctl CLI Reference](https://cert-manager.io/docs/reference/cmctl/)
