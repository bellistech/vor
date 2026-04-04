# Cosign (Container Image Signing)

Cosign is a tool from the Sigstore project for signing, verifying, and attaching metadata to container images and other OCI artifacts, supporting both key-pair and keyless (Fulcio/Rekor) workflows with transparency log integration.

## Installation

### Install Cosign

```bash
# Install via Homebrew
brew install cosign

# Install via Go
go install github.com/sigstore/cosign/v2/cmd/cosign@latest

# Install specific version (Linux amd64)
COSIGN_VERSION=2.4.1
curl -fsSL https://github.com/sigstore/cosign/releases/download/v${COSIGN_VERSION}/cosign-linux-amd64 \
  -o /usr/local/bin/cosign
chmod +x /usr/local/bin/cosign

# Verify installation
cosign version

# Docker-based usage
docker run --rm gcr.io/projectsigstore/cosign version
```

## Key-Pair Signing

### Generate Keys and Sign

```bash
# Generate a key pair (creates cosign.key and cosign.pub)
cosign generate-key-pair

# Generate key pair backed by KMS
cosign generate-key-pair --kms awskms:///alias/cosign-key
cosign generate-key-pair --kms gcpkms://projects/myproj/locations/global/keyRings/kr/cryptoKeys/ck
cosign generate-key-pair --kms hashivault://transit/keys/cosign

# Sign a container image with a key
cosign sign --key cosign.key myregistry.io/myapp:v1.0

# Sign with annotations
cosign sign --key cosign.key \
  -a env=production \
  -a commit=$(git rev-parse HEAD) \
  myregistry.io/myapp:v1.0

# Sign by digest (recommended over tags)
cosign sign --key cosign.key myregistry.io/myapp@sha256:abc123...
```

### Verify Signatures

```bash
# Verify with public key
cosign verify --key cosign.pub myregistry.io/myapp:v1.0

# Verify with KMS key
cosign verify --key awskms:///alias/cosign-key myregistry.io/myapp:v1.0

# Verify with annotations
cosign verify --key cosign.pub \
  -a env=production \
  myregistry.io/myapp:v1.0

# Verify and output JSON
cosign verify --key cosign.pub myregistry.io/myapp:v1.0 | jq .

# Verify multiple images
cosign verify --key cosign.pub \
  myregistry.io/app1:v1 \
  myregistry.io/app2:v1
```

## Keyless Signing (Sigstore)

### Sign with OIDC Identity

```bash
# Keyless sign (opens browser for OIDC auth)
cosign sign myregistry.io/myapp:v1.0

# Keyless sign in CI (uses ambient credentials)
# GitHub Actions: automatic OIDC token
# GitLab CI: SIGSTORE_ID_TOKEN env var
cosign sign myregistry.io/myapp:v1.0

# Force keyless mode
COSIGN_EXPERIMENTAL=1 cosign sign myregistry.io/myapp:v1.0

# Verify keyless signature with identity constraints
cosign verify \
  --certificate-identity user@example.com \
  --certificate-oidc-issuer https://accounts.google.com \
  myregistry.io/myapp:v1.0

# Verify with identity regex
cosign verify \
  --certificate-identity-regexp '.*@example\.com' \
  --certificate-oidc-issuer https://accounts.google.com \
  myregistry.io/myapp:v1.0

# Verify GitHub Actions identity
cosign verify \
  --certificate-identity "https://github.com/myorg/myrepo/.github/workflows/build.yml@refs/heads/main" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  myregistry.io/myapp:v1.0
```

## Transparency Log (Rekor)

### Interact with Rekor

```bash
# Search Rekor for entries by email
rekor-cli search --email user@example.com

# Search by SHA256 of artifact
rekor-cli search --sha sha256:abc123...

# Get a specific log entry
rekor-cli get --uuid 24296fb24b8ad77aed...

# Verify inclusion proof
cosign verify --key cosign.pub \
  --rekor-url https://rekor.sigstore.dev \
  myregistry.io/myapp:v1.0

# Use custom Rekor instance
cosign sign --key cosign.key \
  --rekor-url https://rekor.internal.example.com \
  myregistry.io/myapp:v1.0

# View transparency log entry for a signature
cosign verify --key cosign.pub myregistry.io/myapp:v1.0 2>&1 | jq '.[] | .optional'
```

## SBOM Attachment

### Attach and Verify SBOMs

```bash
# Generate SBOM with syft
syft myregistry.io/myapp:v1.0 -o cyclonedx-json > sbom.cdx.json

# Attach SBOM to image
cosign attach sbom --sbom sbom.cdx.json myregistry.io/myapp:v1.0

# Attach SBOM with specific media type
cosign attach sbom \
  --sbom sbom.cdx.json \
  --type cyclonedx \
  myregistry.io/myapp:v1.0

# Sign the attached SBOM
cosign sign --key cosign.key \
  --attachment sbom \
  myregistry.io/myapp:v1.0

# Verify SBOM signature
cosign verify --key cosign.pub \
  --attachment sbom \
  myregistry.io/myapp:v1.0

# Download attached SBOM
cosign download sbom myregistry.io/myapp:v1.0 > downloaded-sbom.json

# Attest SBOM (in-toto attestation)
cosign attest --key cosign.key \
  --predicate sbom.cdx.json \
  --type cyclonedx \
  myregistry.io/myapp:v1.0
```

## Attestation and Policy

### In-Toto Attestations

```bash
# Create a custom attestation
cosign attest --key cosign.key \
  --predicate provenance.json \
  --type slsaprovenance \
  myregistry.io/myapp:v1.0

# Verify attestation
cosign verify-attestation --key cosign.pub \
  --type slsaprovenance \
  myregistry.io/myapp:v1.0

# Verify attestation with policy (Rego/CUE)
cosign verify-attestation --key cosign.pub \
  --type cyclonedx \
  --policy policy.cue \
  myregistry.io/myapp:v1.0

# Keyless attestation verification
cosign verify-attestation \
  --certificate-identity user@example.com \
  --certificate-oidc-issuer https://accounts.google.com \
  --type slsaprovenance \
  myregistry.io/myapp:v1.0
```

## GitHub Actions Integration

### Workflow Configuration

```yaml
# .github/workflows/sign.yml
jobs:
  sign:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      id-token: write       # Required for keyless signing
      packages: write
    steps:
      - uses: sigstore/cosign-installer@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Sign image
        run: |
          cosign sign ghcr.io/${{ github.repository }}@${{ steps.build.outputs.digest }}
      - name: Verify signature
        run: |
          cosign verify \
            --certificate-identity "https://github.com/${{ github.repository }}/.github/workflows/sign.yml@${{ github.ref }}" \
            --certificate-oidc-issuer https://token.actions.githubusercontent.com \
            ghcr.io/${{ github.repository }}@${{ steps.build.outputs.digest }}
```

## Policy Enforcement

### Kubernetes Admission Control

```yaml
# Sigstore Policy Controller - ClusterImagePolicy
apiVersion: policy.sigstore.dev/v1beta1
kind: ClusterImagePolicy
metadata:
  name: require-signed-images
spec:
  images:
    - glob: "myregistry.io/**"
  authorities:
    - key:
        data: |
          -----BEGIN PUBLIC KEY-----
          MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE...
          -----END PUBLIC KEY-----
    - keyless:
        url: https://fulcio.sigstore.dev
        identities:
          - issuer: https://token.actions.githubusercontent.com
            subject: "https://github.com/myorg/*"
```

## Tips

- Always sign by digest (`@sha256:...`) rather than tag to prevent tag mutation attacks
- Use keyless signing in CI/CD pipelines to avoid managing long-lived signing keys
- Store cosign private keys in KMS (AWS KMS, GCP KMS, HashiCorp Vault) rather than on disk
- Verify both `--certificate-identity` and `--certificate-oidc-issuer` in keyless verification to prevent identity spoofing
- Attach SBOMs as signed attestations (`cosign attest`) rather than plain attachments for tamper evidence
- Use the Sigstore Policy Controller or Kyverno for Kubernetes admission enforcement of signed images
- Check Rekor transparency log entries to audit the signing history of your images
- Pin cosign versions in CI to avoid breaking changes during minor releases
- Use `cosign tree` to inspect all signatures, attestations, and SBOMs attached to an image
- Rotate signing keys periodically and use key versioning with KMS-backed keys
- Set `COSIGN_REPOSITORY` to store signatures in a separate OCI repository when the image registry is read-only

## See Also

- container-security, sbom, grype, trivy, pki, opa

## References

- [Cosign Documentation](https://docs.sigstore.dev/cosign/overview/)
- [Sigstore Project](https://www.sigstore.dev/)
- [Fulcio Certificate Authority](https://docs.sigstore.dev/fulcio/overview/)
- [Rekor Transparency Log](https://docs.sigstore.dev/rekor/overview/)
- [Sigstore Policy Controller](https://docs.sigstore.dev/policy-controller/overview/)
- [SLSA Provenance Specification](https://slsa.dev/provenance/)
