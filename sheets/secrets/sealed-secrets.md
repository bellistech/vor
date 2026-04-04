# Sealed Secrets (Kubernetes Encrypted Secrets)

Kubernetes controller and CLI tool (kubeseal) that encrypts secrets into SealedSecret resources safe to store in git, decryptable only by the controller running in the target cluster.

## Installation

### Install kubeseal CLI

```bash
# macOS via Homebrew
brew install kubeseal

# Linux (download binary)
KUBESEAL_VERSION=0.27.0
curl -OL "https://github.com/bitnami-labs/sealed-secrets/releases/download/v${KUBESEAL_VERSION}/kubeseal-${KUBESEAL_VERSION}-linux-amd64.tar.gz"
tar -xvzf kubeseal-${KUBESEAL_VERSION}-linux-amd64.tar.gz kubeseal
sudo install -m 755 kubeseal /usr/local/bin/kubeseal

# Verify version
kubeseal --version
```

### Install controller in cluster

```bash
# Install via Helm
helm repo add sealed-secrets https://bitnami-labs.github.io/sealed-secrets
helm repo update
helm install sealed-secrets sealed-secrets/sealed-secrets \
  --namespace kube-system \
  --set fullnameOverride=sealed-secrets-controller

# Or install via kubectl
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.27.0/controller.yaml

# Verify controller is running
kubectl get pods -n kube-system -l app.kubernetes.io/name=sealed-secrets

# Fetch the public certificate (for offline sealing)
kubeseal --fetch-cert > pub-cert.pem
```

## Basic Usage

### Create sealed secrets

```bash
# Method 1: Pipe a K8s Secret through kubeseal
kubectl create secret generic db-creds \
  --from-literal=username=admin \
  --from-literal=password=s3cur3p@ss \
  --dry-run=client -o yaml | \
  kubeseal --format yaml > db-creds-sealed.yaml

# Method 2: Seal from an existing secret YAML file
kubeseal --format yaml < secret.yaml > sealed-secret.yaml

# Method 3: Use offline mode with a saved certificate
kubeseal --cert pub-cert.pem --format yaml < secret.yaml > sealed-secret.yaml

# Method 4: Seal individual values (raw mode)
echo -n "s3cur3p@ss" | kubeseal --raw \
  --namespace default --name db-creds \
  --from-file=/dev/stdin

# Output as JSON (default)
kubeseal < secret.yaml > sealed-secret.json
```

### Apply sealed secrets

```bash
# Apply the sealed secret to the cluster
kubectl apply -f db-creds-sealed.yaml

# The controller decrypts it into a regular Secret
kubectl get secret db-creds

# Verify the sealed secret status
kubectl get sealedsecret db-creds -o yaml

# View the decrypted secret
kubectl get secret db-creds -o jsonpath='{.data.password}' | base64 -d
```

### SealedSecret resource structure

```yaml
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  name: db-creds
  namespace: default
spec:
  encryptedData:
    username: AgBy3i4OJSWK+PiTySYZ...  # RSA-OAEP encrypted, base64
    password: AgBjE2k9JXHZ+PfQ8Za...
  template:
    metadata:
      name: db-creds
      namespace: default
    type: Opaque
```

## Scoping

### Namespace-scoped (default — strict)

```bash
# Secret is bound to namespace + name (default, most secure)
kubectl create secret generic api-key \
  --from-literal=key=abc123 \
  --dry-run=client -o yaml | \
  kubeseal --scope strict --format yaml > api-key-sealed.yaml

# Cannot be moved to another namespace or renamed
```

### Namespace-wide scope

```bash
# Secret can be used with any name within the same namespace
kubectl create secret generic api-key \
  --from-literal=key=abc123 \
  --dry-run=client -o yaml | \
  kubeseal --scope namespace-wide --format yaml > api-key-sealed.yaml
```

### Cluster-wide scope

```bash
# Secret can be decrypted in any namespace with any name
kubectl create secret generic api-key \
  --from-literal=key=abc123 \
  --dry-run=client -o yaml | \
  kubeseal --scope cluster-wide --format yaml > api-key-sealed.yaml
```

### Scope comparison

```bash
# strict (default):    bound to exact namespace + name
# namespace-wide:      bound to namespace, any name
# cluster-wide:        usable anywhere in the cluster
```

## Certificate Management

### Fetch and backup certificates

```bash
# Fetch current public certificate
kubeseal --fetch-cert > pub-cert.pem

# Fetch from a specific controller
kubeseal --fetch-cert \
  --controller-name=sealed-secrets-controller \
  --controller-namespace=kube-system > pub-cert.pem

# View certificate details
openssl x509 -in pub-cert.pem -noout -text

# Check certificate expiry
openssl x509 -in pub-cert.pem -noout -enddate
```

### Backup sealing keys

```bash
# Backup the controller's private key (CRITICAL — store securely)
kubectl get secret -n kube-system \
  -l sealedsecrets.bitnami.com/sealed-secrets-key \
  -o yaml > sealed-secrets-keys-backup.yaml

# List all sealing keys
kubectl get secret -n kube-system \
  -l sealedsecrets.bitnami.com/sealed-secrets-key

# Restore keys (before reinstalling controller)
kubectl apply -f sealed-secrets-keys-backup.yaml
```

### Certificate rotation

```bash
# Controller auto-rotates every 30 days by default
# Old keys are kept for decryption; new keys used for sealing

# Force key renewal
kubectl annotate secret -n kube-system \
  -l sealedsecrets.bitnami.com/sealed-secrets-key \
  sealedsecrets.bitnami.com/sealed-secrets-key-renewal=true

# Configure rotation period via Helm
helm upgrade sealed-secrets sealed-secrets/sealed-secrets \
  --namespace kube-system \
  --set keyRenewPeriod=720h  # 30 days

# Re-seal all secrets with the new certificate
kubeseal --fetch-cert > new-cert.pem
kubeseal --cert new-cert.pem < old-sealed.yaml > new-sealed.yaml
```

## Updating Sealed Secrets

### Add or update individual values

```bash
# Merge a new key into an existing sealed secret
echo -n "new-value" | kubeseal --raw \
  --namespace default --name db-creds \
  --merge-into db-creds-sealed.yaml \
  --from-file=/dev/stdin

# Update with a label indicating the key name
echo -n "new-value" | kubeseal --raw \
  --namespace default --name db-creds \
  --from-file=/dev/stdin >> add-to-sealed.yaml
```

### Replace entire sealed secret

```bash
# Recreate from updated source secret
kubectl create secret generic db-creds \
  --from-literal=username=admin \
  --from-literal=password=new-p@ss \
  --from-literal=host=db.example.com \
  --dry-run=client -o yaml | \
  kubeseal --format yaml > db-creds-sealed.yaml

kubectl apply -f db-creds-sealed.yaml
```

## Template Customization

### Secret metadata and type

```yaml
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  name: tls-cert
  namespace: default
spec:
  encryptedData:
    tls.crt: AgBy3i4OJSWK+PiT...
    tls.key: AgBjE2k9JXHZ+PfQ...
  template:
    metadata:
      name: tls-cert
      namespace: default
      labels:
        app: web
        managed-by: sealed-secrets
      annotations:
        description: "TLS certificate for web app"
    type: kubernetes.io/tls
```

### Docker registry secret

```bash
# Create a docker registry sealed secret
kubectl create secret docker-registry regcred \
  --docker-server=registry.example.com \
  --docker-username=user \
  --docker-password=p@ssw0rd \
  --dry-run=client -o yaml | \
  kubeseal --format yaml > regcred-sealed.yaml
```

## GitOps Workflow

### ArgoCD / Flux integration

```bash
# Sealed secrets work natively with GitOps
# 1. Developer seals a secret locally
kubeseal --cert pub-cert.pem --format yaml < secret.yaml > sealed-secret.yaml

# 2. Commit sealed secret to git
git add sealed-secret.yaml
git commit -m "Add sealed database credentials"
git push

# 3. GitOps controller syncs SealedSecret to cluster
# 4. Sealed secrets controller decrypts to regular Secret
# 5. Application pods consume the Secret normally

# Verify sync status (ArgoCD)
# argocd app get my-app --show-operation
```

## Tips

- Always back up the controller's private keys — losing them means all sealed secrets become undecryptable.
- Use `strict` scope (default) unless you have a specific reason for namespace-wide or cluster-wide.
- Fetch and cache the public certificate for offline sealing in CI/CD pipelines.
- Sealed secrets are one-way: you cannot decrypt a SealedSecret YAML without the controller's private key.
- The controller keeps old keys during rotation, so existing sealed secrets continue to work.
- Use `--merge-into` to add individual keys without re-sealing the entire secret.
- Store the public certificate (`pub-cert.pem`) in your git repo — it is not sensitive.
- Sealed secrets are cluster-specific by default; you cannot move them between clusters without re-sealing.
- Combine with ArgoCD or Flux for a complete GitOps secrets workflow.
- Use template metadata to add labels and annotations that will appear on the decrypted Secret.
- Monitor the controller logs for decryption errors: `kubectl logs -n kube-system deploy/sealed-secrets-controller`.
- Test sealed secrets in a staging cluster before applying to production.

## See Also

sops, iam, kubernetes, vault, helm

## References

- [Sealed Secrets GitHub Repository](https://github.com/bitnami-labs/sealed-secrets)
- [Sealed Secrets Helm Chart](https://github.com/bitnami-labs/sealed-secrets/tree/main/helm/sealed-secrets)
- [Kubernetes Secrets Documentation](https://kubernetes.io/docs/concepts/configuration/secret/)
- [Sealed Secrets FAQ](https://github.com/bitnami-labs/sealed-secrets/blob/main/docs/FAQ.md)
- [Bitnami Sealed Secrets Blog](https://engineering.bitnami.com/articles/sealed-secrets.html)
