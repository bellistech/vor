# SOPS (Secrets OPerationS)

Editor-friendly encrypted file tool that encrypts only values (not keys) in YAML, JSON, ENV, and INI files, supporting age, PGP, AWS KMS, GCP KMS, and Azure Key Vault backends with git-friendly diffs.

## Installation

### Install sops

```bash
# macOS via Homebrew
brew install sops

# Linux (download binary)
curl -LO https://github.com/getsops/sops/releases/download/v3.9.0/sops-v3.9.0.linux.amd64
chmod +x sops-v3.9.0.linux.amd64
sudo mv sops-v3.9.0.linux.amd64 /usr/local/bin/sops

# Verify installation
sops --version
```

### Install age (recommended backend)

```bash
# macOS
brew install age

# Linux
apt install age

# Generate a key pair
age-keygen -o ~/.config/sops/age/keys.txt

# View your public key
age-keygen -y ~/.config/sops/age/keys.txt
# age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
```

## Basic Usage

### Encrypt files

```bash
# Encrypt a YAML file with age
sops encrypt --age age1ql3z7hjy... secrets.yaml > secrets.enc.yaml

# Encrypt in-place
sops encrypt -i --age age1ql3z7hjy... secrets.yaml

# Encrypt with PGP
sops encrypt --pgp FINGERPRINT secrets.yaml > secrets.enc.yaml

# Encrypt with AWS KMS
sops encrypt --kms arn:aws:kms:us-east-1:123456789012:key/key-id \
  secrets.yaml > secrets.enc.yaml

# Encrypt with GCP KMS
sops encrypt \
  --gcp-kms projects/my-project/locations/global/keyRings/my-ring/cryptoKeys/my-key \
  secrets.yaml > secrets.enc.yaml

# Encrypt a JSON file
sops encrypt --age age1ql3z7hjy... config.json > config.enc.json

# Encrypt a .env file
sops encrypt --age age1ql3z7hjy... .env > .env.enc
```

### Decrypt files

```bash
# Decrypt to stdout
sops decrypt secrets.enc.yaml

# Decrypt in-place
sops decrypt -i secrets.enc.yaml

# Decrypt to a specific file
sops decrypt secrets.enc.yaml > secrets.yaml

# Decrypt with specific key file
SOPS_AGE_KEY_FILE=~/.config/sops/age/keys.txt sops decrypt secrets.enc.yaml

# Extract a single value
sops decrypt --extract '["database"]["password"]' secrets.enc.yaml
```

### Edit encrypted files

```bash
# Open encrypted file in $EDITOR (decrypts, edits, re-encrypts)
sops secrets.enc.yaml

# Edit with a specific editor
EDITOR=vim sops secrets.enc.yaml

# Set values without opening editor
sops set secrets.enc.yaml '["database"]["password"]' '"new-password"'
```

## Configuration (.sops.yaml)

### Project-level configuration

```yaml
# .sops.yaml — placed in repo root
creation_rules:
  # Production secrets — encrypted with KMS + age
  - path_regex: secrets/prod/.*\.yaml$
    kms: arn:aws:kms:us-east-1:123456789012:key/prod-key
    age: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p

  # Staging secrets — age only
  - path_regex: secrets/staging/.*\.yaml$
    age: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p

  # Terraform variables
  - path_regex: \.tfvars$
    age: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p

  # Default rule for everything else
  - age: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
```

### Encrypted key groups (multi-key)

```yaml
# Require keys from multiple groups (M-of-N)
creation_rules:
  - path_regex: secrets/critical/.*
    shamir_threshold: 2  # Need 2 of 3 groups
    key_groups:
      - age:
          - age1ql3z7hjy...  # DevOps team
      - age:
          - age1abc123...    # Security team
      - kms:
          - arn: arn:aws:kms:us-east-1:123456789012:key/backup-key
```

### Encrypted suffix / regex

```yaml
# Only encrypt specific keys
creation_rules:
  - path_regex: .*\.yaml$
    encrypted_suffix: _secret
    age: age1ql3z7hjy...
```

```yaml
# Example: only fields ending in _secret are encrypted
database:
  host: db.example.com           # NOT encrypted (plaintext)
  port: 5432                     # NOT encrypted (plaintext)
  password_secret: ENC[AES256...]  # Encrypted
  api_key_secret: ENC[AES256...]   # Encrypted
```

## Multiple Backends

### AWS KMS

```bash
# Encrypt with KMS (uses AWS credentials from env/profile)
sops encrypt --kms arn:aws:kms:us-east-1:123456789012:key/my-key secrets.yaml

# Multiple KMS keys (cross-region for DR)
sops encrypt \
  --kms "arn:aws:kms:us-east-1:123456789012:key/key1,arn:aws:kms:eu-west-1:123456789012:key/key2" \
  secrets.yaml

# KMS with assumed role
sops encrypt \
  --kms "arn:aws:kms:us-east-1:123456789012:key/key1+arn:aws:iam::123456789012:role/sops-role" \
  secrets.yaml
```

### GCP KMS

```bash
# Encrypt with GCP KMS
sops encrypt \
  --gcp-kms projects/my-project/locations/global/keyRings/sops/cryptoKeys/sops-key \
  secrets.yaml
```

### Azure Key Vault

```bash
# Encrypt with Azure Key Vault
sops encrypt \
  --azure-kv https://my-vault.vault.azure.net/keys/sops-key/abc123 \
  secrets.yaml
```

### HashiCorp Vault Transit

```bash
# Encrypt with Vault transit
sops encrypt \
  --hc-vault-transit https://vault.example.com/v1/transit/keys/sops \
  secrets.yaml
```

## Git Integration

### Diff encrypted files

```bash
# .gitattributes — show decrypted diffs
*.enc.yaml diff=sopsdiffer

# .git/config or global gitconfig
# [diff "sopsdiffer"]
#   textconv = sops decrypt
```

```bash
# Configure git diff driver
git config diff.sopsdiffer.textconv "sops decrypt"

# Now git diff shows decrypted content
git diff secrets.enc.yaml
```

### Pre-commit hooks

```bash
# .pre-commit-config.yaml
# - repo: https://github.com/getsops/sops
#   hooks:
#     - id: sops-check
#       name: Check for unencrypted secrets
```

```bash
# Simple pre-commit hook to prevent committing unencrypted secrets
cat > .git/hooks/pre-commit << 'HOOK'
#!/bin/bash
for f in $(git diff --cached --name-only | grep -E '\.(yaml|json|env)$' | grep -v '\.enc\.'); do
  if grep -q 'password\|secret\|api_key' "$f" 2>/dev/null; then
    echo "ERROR: $f may contain unencrypted secrets"
    exit 1
  fi
done
HOOK
chmod +x .git/hooks/pre-commit
```

## Key Rotation

### Rotate data keys

```bash
# Rotate the data key (re-encrypts with same master keys)
sops rotate -i secrets.enc.yaml

# Rotate and add a new recipient
sops rotate -i --add-age age1newkey... secrets.enc.yaml

# Rotate and remove a recipient
sops rotate -i --rm-age age1oldkey... secrets.enc.yaml

# Update master keys from .sops.yaml
sops updatekeys secrets.enc.yaml
```

### Batch rotation

```bash
# Rotate all encrypted files
find . -name "*.enc.yaml" -exec sops rotate -i {} \;

# Update keys for all files matching .sops.yaml rules
find . -name "*.enc.yaml" -exec sops updatekeys -y {} \;
```

## CI/CD Integration

### Use in pipelines

```bash
# GitHub Actions — decrypt with age key from secret
export SOPS_AGE_KEY=${{ secrets.SOPS_AGE_KEY }}
sops decrypt secrets.enc.yaml > secrets.yaml

# Decrypt and export as env vars
eval $(sops decrypt --output-type dotenv secrets.enc.yaml)

# Decrypt and pipe to kubectl
sops decrypt k8s-secrets.enc.yaml | kubectl apply -f -

# Use with Helm
sops decrypt values.enc.yaml | helm upgrade --install my-app ./chart -f -

# Use with Terraform
sops decrypt terraform.enc.tfvars > terraform.tfvars
terraform apply -var-file=terraform.tfvars
rm terraform.tfvars
```

## Tips

- Always use `.sops.yaml` in your repo root to avoid passing key arguments on every command.
- Prefer `age` over PGP for new projects — simpler key management, no keyring complexity.
- Use `encrypted_suffix` or `encrypted_regex` to encrypt only sensitive fields, keeping non-secret config readable.
- SOPS encrypts values but leaves keys in plaintext, making git diffs meaningful and reviewable.
- Use Shamir threshold with multiple key groups for critical secrets requiring multi-party access.
- Store age private keys in a password manager or secure vault, never in the repository.
- Set `SOPS_AGE_KEY_FILE` environment variable to avoid specifying the key file every time.
- Use `sops decrypt --extract` to pull a single value for scripting without decrypting the entire file.
- Run `sops updatekeys` after changing `.sops.yaml` rules to apply new key configurations to existing files.
- Combine cloud KMS (for team/service access) with age keys (for developer access) for flexible multi-backend setups.
- Always clean up decrypted files in CI/CD pipelines (use traps or temp directories).
- Use `sops publish` to push encrypted files to S3/GCS for centralized secret distribution.

## See Also

sealed-secrets, iam, age, gpg, vault

## References

- [SOPS GitHub Repository](https://github.com/getsops/sops)
- [SOPS Documentation](https://getsops.io/)
- [age Encryption Tool](https://github.com/FiloSottile/age)
- [AWS KMS Documentation](https://docs.aws.amazon.com/kms/latest/developerguide/)
- [GCP Cloud KMS](https://cloud.google.com/kms/docs)
- [Azure Key Vault](https://learn.microsoft.com/en-us/azure/key-vault/)
