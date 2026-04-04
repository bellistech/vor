# external-secrets

Kubernetes operator that synchronizes secrets from external providers into native Kubernetes Secret objects.

## Installation

```bash
# Install via Helm
helm repo add external-secrets https://charts.external-secrets.io
helm repo update
helm install external-secrets external-secrets/external-secrets \
  --namespace external-secrets \
  --create-namespace

# Verify installation
kubectl get pods -n external-secrets
kubectl get crd | grep external-secrets
```

## SecretStore and ClusterSecretStore

```bash
# AWS Secrets Manager SecretStore (namespace-scoped)
cat <<EOF | kubectl apply -f -
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: aws-secretsmanager
  namespace: default
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      auth:
        secretRef:
          accessKeyIDSecretRef:
            name: aws-credentials
            key: access-key-id
          secretAccessKeySecretRef:
            name: aws-credentials
            key: secret-access-key
EOF

# HashiCorp Vault ClusterSecretStore
cat <<EOF | kubectl apply -f -
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: vault-store
spec:
  provider:
    vault:
      server: https://vault.example.com
      path: secret
      version: v2
      auth:
        kubernetes:
          mountPath: kubernetes
          role: external-secrets
          serviceAccountRef:
            name: external-secrets
            namespace: external-secrets
EOF

# GCP Secret Manager ClusterSecretStore
cat <<EOF | kubectl apply -f -
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: gcp-secretmanager
spec:
  provider:
    gcpsm:
      projectID: my-gcp-project
      auth:
        secretRef:
          secretAccessKeySecretRef:
            name: gcp-sa-key
            key: credentials.json
            namespace: external-secrets
EOF

# Azure Key Vault ClusterSecretStore
cat <<EOF | kubectl apply -f -
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: azure-keyvault
spec:
  provider:
    azurekv:
      tenantId: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      vaultUrl: "https://my-vault.vault.azure.net"
      authType: ManagedIdentity
      identityId: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
EOF
```

## ExternalSecret Resources

```bash
# Basic ExternalSecret syncing a single key
cat <<EOF | kubectl apply -f -
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: db-credentials
  namespace: default
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secretsmanager
    kind: SecretStore
  target:
    name: db-credentials
    creationPolicy: Owner
  data:
  - secretKey: username
    remoteRef:
      key: prod/database
      property: username
  - secretKey: password
    remoteRef:
      key: prod/database
      property: password
EOF

# Sync entire secret as JSON using dataFrom
cat <<EOF | kubectl apply -f -
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: app-config
  namespace: default
spec:
  refreshInterval: 30m
  secretStoreRef:
    name: vault-store
    kind: ClusterSecretStore
  target:
    name: app-config
  dataFrom:
  - extract:
      key: secret/data/app/config
EOF

# Using find to discover secrets by tags or name pattern
cat <<EOF | kubectl apply -f -
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: tagged-secrets
  namespace: default
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secretsmanager
    kind: SecretStore
  target:
    name: tagged-secrets
  dataFrom:
  - find:
      tags:
        environment: production
        team: backend
EOF
```

## Templating

```bash
# ExternalSecret with Go template for custom formatting
cat <<EOF | kubectl apply -f -
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: db-connection-string
  namespace: default
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secretsmanager
    kind: SecretStore
  target:
    name: db-connection-string
    template:
      type: Opaque
      data:
        connection_string: "postgresql://{{ .username }}:{{ .password }}@{{ .host }}:5432/{{ .dbname }}?sslmode=require"
      metadata:
        labels:
          app: myapp
        annotations:
          managed-by: external-secrets
  data:
  - secretKey: username
    remoteRef:
      key: prod/database
      property: username
  - secretKey: password
    remoteRef:
      key: prod/database
      property: password
  - secretKey: host
    remoteRef:
      key: prod/database
      property: host
  - secretKey: dbname
    remoteRef:
      key: prod/database
      property: dbname
EOF

# Template with engineVersion v2 and functions
cat <<EOF | kubectl apply -f -
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: tls-cert
  namespace: default
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-store
    kind: ClusterSecretStore
  target:
    name: tls-cert
    template:
      engineVersion: v2
      type: kubernetes.io/tls
      data:
        tls.crt: "{{ .cert | b64dec }}"
        tls.key: "{{ .key | b64dec }}"
  data:
  - secretKey: cert
    remoteRef:
      key: secret/data/tls/example
      property: certificate
  - secretKey: key
    remoteRef:
      key: secret/data/tls/example
      property: private_key
EOF
```

## PushSecret

```bash
# Push a Kubernetes Secret to an external provider
cat <<EOF | kubectl apply -f -
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-db-creds
  namespace: default
spec:
  updatePolicy: Replace
  deletionPolicy: Delete
  refreshInterval: 10m
  secretStoreRefs:
  - name: aws-secretsmanager
    kind: SecretStore
  selector:
    secret:
      name: db-credentials
  data:
  - match:
      secretKey: username
      remoteRef:
        remoteKey: prod/database
        property: username
  - match:
      secretKey: password
      remoteRef:
        remoteKey: prod/database
        property: password
EOF
```

## Troubleshooting

```bash
# Check ExternalSecret sync status
kubectl get externalsecret -A
kubectl describe externalsecret db-credentials

# Check SecretStore connectivity
kubectl get secretstore -A -o wide
kubectl describe secretstore aws-secretsmanager

# View operator logs
kubectl logs -n external-secrets deploy/external-secrets -f

# Check conditions on ExternalSecret
kubectl get externalsecret db-credentials -o jsonpath='{.status.conditions}'

# Verify the generated Kubernetes Secret
kubectl get secret db-credentials -o yaml

# Force immediate sync by deleting the owned secret
kubectl delete secret db-credentials
# The operator will recreate it on next reconciliation
```

## Tips

- Set `refreshInterval` based on how often secrets rotate; 1h is a sensible default for most workloads
- Use `ClusterSecretStore` for shared provider configs and `SecretStore` for team-isolated credentials
- The `creationPolicy: Owner` ensures the Secret is garbage-collected when the ExternalSecret is deleted
- Use `dataFrom.extract` to sync all keys from a single remote secret without listing them individually
- Template engine v2 supports Sprig functions for base64 decoding, string manipulation, and conditionals
- PushSecret enables GitOps workflows where secrets originate in-cluster and sync outward
- Set resource limits on the operator pods to prevent memory issues with large secret volumes
- Use `find` with tags to dynamically discover secrets without hardcoding remote key paths
- Monitor the `externalsecret_status_condition` Prometheus metric for sync failure alerts
- Combine with cert-manager: store ACME account keys in Vault, reference via ExternalSecret
- Always use IRSA (AWS), Workload Identity (GCP), or Pod Identity (Azure) over static credentials
- Test SecretStore connectivity before creating ExternalSecrets by checking store status conditions

## See Also

- cert-manager
- opa
- kyverno

## References

- [External Secrets Operator Documentation](https://external-secrets.io/latest/)
- [External Secrets GitHub Repository](https://github.com/external-secrets/external-secrets)
- [AWS Secrets Manager Provider](https://external-secrets.io/latest/provider/aws-secrets-manager/)
- [HashiCorp Vault Provider](https://external-secrets.io/latest/provider/hashicorp-vault/)
- [PushSecret API Reference](https://external-secrets.io/latest/api/pushsecret/)
