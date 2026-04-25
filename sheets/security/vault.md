# Hashicorp Vault

Identity-based secrets and encryption management. Centralizes credentials, certificates, encryption keys, and dynamic secrets behind a single API with audit logs and fine-grained policies. Sealed-by-default; cryptographically split master key; supports static KV and dynamic credentials with TTL-bound leases.

## Setup

### Install on macOS (Homebrew)

```bash
# Tap official HashiCorp tap
brew tap hashicorp/tap

# Install OSS edition
brew install hashicorp/tap/vault

# Install Enterprise (separate formula)
brew install hashicorp/tap/vault-enterprise

# Pin a specific version
brew install hashicorp/tap/vault@1.15

# Upgrade
brew upgrade hashicorp/tap/vault

# Uninstall (keeps data)
brew uninstall vault
```

### Install on Debian/Ubuntu (APT)

```bash
# Add GPG key
wget -O- https://apt.releases.hashicorp.com/gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg

# Add repo
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] \
  https://apt.releases.hashicorp.com $(lsb_release -cs) main" \
  | sudo tee /etc/apt/sources.list.d/hashicorp.list

sudo apt-get update
sudo apt-get install vault                # OSS
sudo apt-get install vault-enterprise     # Enterprise

# Pin a version
sudo apt-get install vault=1.15.4-1
```

### Install on RHEL/CentOS/Fedora (YUM/DNF)

```bash
sudo yum install -y yum-utils
sudo yum-config-manager --add-repo https://rpm.releases.hashicorp.com/RHEL/hashicorp.repo
sudo yum -y install vault
# or
sudo dnf install -y vault
```

### Install via Binary (Linux/macOS/Windows)

```bash
# Download for current platform/arch
VAULT_VERSION=1.15.4
curl -fsSLO "https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_linux_amd64.zip"

# Verify with HashiCorp signing key (recommended)
curl -fsSLO "https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_SHA256SUMS"
curl -fsSLO "https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_SHA256SUMS.sig"
gpg --verify vault_${VAULT_VERSION}_SHA256SUMS.sig vault_${VAULT_VERSION}_SHA256SUMS
shasum -a 256 -c vault_${VAULT_VERSION}_SHA256SUMS --ignore-missing

unzip vault_${VAULT_VERSION}_linux_amd64.zip
sudo install -m 0755 vault /usr/local/bin/vault
vault --version
```

### Dev Mode (in-memory, root token preset)

```bash
# Single in-memory server, unsealed at start, prints root token
vault server -dev

# With a custom root token
vault server -dev -dev-root-token-id=root

# Bind to all interfaces (DEV ONLY)
vault server -dev -dev-listen-address=0.0.0.0:8200

# With a TLS cert (still dev backend)
vault server -dev -dev-tls

# After dev start, set environment in another terminal
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='root'
vault status
```

DEV MODE WARNING: dev mode keeps all data in memory and ships an unsealed Vault with a fixed root token. Never run in production.

### OSS vs Enterprise

OSS (Community Edition) ships KV v1/v2, transit, PKI, SSH, database, AWS/GCP/Azure/Kubernetes secret engines, all OSS auth methods, Raft integrated storage, audit devices, identity, agent, and policies.

Enterprise adds: namespaces, performance and DR replication, HSM auto-unseal via PKCS#11, MFA, control groups, sentinel policies, secrets sync, lease count quotas, performance standby nodes, KMIP, transform secret engine (FPE/tokenization/masking), seal-wrap, FIPS 140-2 builds, eventual consistency cache, and adaptive overload protection.

### Version Commands

```bash
vault version
# Vault v1.15.4 (...) Edition: ce | ent | ent.hsm

vault status
# Includes Version, Build Date, Storage Type, HA Mode, Active Node Address

vault read sys/health
# JSON: version, cluster_name, cluster_id, sealed, standby, performance_standby, replication_*

vault read sys/seal-status
# Type: shamir|awskms|... ; Initialized; Sealed; Threshold; N
```

## Environment Variables

### Connection

```bash
export VAULT_ADDR=https://vault.example.com:8200   # API endpoint (REQUIRED)
export VAULT_TOKEN=hvs.CAESI...                    # Auth token; falls back to ~/.vault-token
export VAULT_NAMESPACE=team-a                      # Enterprise namespace prefix (X-Vault-Namespace)
```

### TLS / mTLS

```bash
export VAULT_CACERT=/etc/vault.d/ca.pem            # PEM bundle to verify Vault cert
export VAULT_CAPATH=/etc/vault.d/ca/                # Directory of PEM CA certs
export VAULT_CLIENT_CERT=/etc/vault.d/client.pem   # Client cert for mTLS auth (cert method)
export VAULT_CLIENT_KEY=/etc/vault.d/client-key.pem
export VAULT_TLS_SERVER_NAME=vault.example.com     # Override SNI when host != cert CN/SAN
export VAULT_SKIP_VERIFY=true                      # DANGEROUS — skip TLS verification
```

### Output / Behavior

```bash
export VAULT_FORMAT=table     # default; human-readable
export VAULT_FORMAT=json      # machine-readable, jq-friendly
export VAULT_FORMAT=yaml      # YAML
export VAULT_FORMAT=raw       # raw bytes (rare)

export VAULT_LICENSE=02MV4UU43BK5HG...  # Enterprise license string at server start
export VAULT_LICENSE_PATH=/etc/vault.d/license.hclic

export VAULT_DISABLE_REDIRECTS=true     # Don't follow standby->active redirect
export VAULT_MAX_RETRIES=2              # Client retries on 5xx
export VAULT_CLIENT_TIMEOUT=60          # Seconds
export VAULT_RATE_LIMIT=10              # Requests/sec from CLI
export VAULT_HTTP_PROXY=http://proxy:3128
export VAULT_PROXY_ADDR=http://proxy:3128
export VAULT_LOG_LEVEL=trace            # CLI debug
export VAULT_CLI_NO_COLOR=true          # Disable ANSI
export VAULT_NO_VERIFY_HOSTNAME=true    # Skip hostname check (still verifies CA)
export VAULT_SRV_LOOKUP=true            # DNS SRV record discovery
```

### Token Helpers

```bash
# Default: token persisted to ~/.vault-token
vault login -method=userpass username=alice
cat ~/.vault-token

# Disable persistence: pass token only via env
unset VAULT_TOKEN
rm ~/.vault-token

# Custom token helper (script that reads/writes/erases)
echo 'token_helper = "/usr/local/bin/vault-token-helper"' >> ~/.vault
```

## Architecture

### Storage Backends

```bash
# Raft (integrated storage) — DEFAULT, embedded, no external dependency
storage "raft" {
  path    = "/opt/vault/data"
  node_id = "vault-1"
  retry_join {
    leader_api_addr = "https://vault-2:8200"
  }
}

# Consul — legacy default before 1.4
storage "consul" {
  address = "127.0.0.1:8500"
  path    = "vault/"
  scheme  = "https"
  token   = "..."
}

# File — single node, dev / homelab
storage "file" {
  path = "/opt/vault/data"
}

# Inmem — TEST ONLY, lost on restart
storage "inmem" {}
```

### Sealed vs Unsealed

Vault starts SEALED — encrypted storage, API rejects most calls. Unseal reconstructs the master key from key shards (Shamir) or unwraps it via an external KMS (auto-unseal). Once unsealed, the master key decrypts the encryption key used for all stored data.

```bash
vault status
# Sealed       true
# Total Shares 5
# Threshold    3
# Unseal Progress 0/3
```

### Seal Types

```bash
# Shamir (default) — split master key into N shards, K to reconstruct
seal "shamir" {}    # implicit; no config needed

# AWS KMS
seal "awskms" {
  region     = "us-east-1"
  kms_key_id = "alias/vault-unseal"
}

# Azure Key Vault
seal "azurekeyvault" {
  tenant_id  = "..."
  vault_name = "vault-unseal"
  key_name   = "vault-key"
}

# GCP KMS
seal "gcpckms" {
  project    = "my-project"
  region     = "global"
  key_ring   = "vault"
  crypto_key = "vault-unseal"
}

# OCI KMS
seal "ocikms" {
  key_id        = "ocid1.key..."
  crypto_endpoint = "..."
  management_endpoint = "..."
}

# Transit (another Vault as the "unsealer")
seal "transit" {
  address    = "https://vault-master:8200"
  token      = "..."
  key_name   = "autounseal"
  mount_path = "transit/"
}

# HSM via PKCS#11 (Enterprise)
seal "pkcs11" {
  lib            = "/usr/safenet/lunaclient/lib/libCryptoki2_64.so"
  slot           = "0"
  pin            = "..."
  key_label      = "vault-hsm-key"
  hmac_key_label = "vault-hsm-hmac"
}
```

### Recovery vs Unseal Keys

With Shamir (manual unseal): operators receive UNSEAL keys. Threshold of these recombines the master key on every start.

With auto-unseal: operators receive RECOVERY keys instead. They never unseal — KMS does. Recovery keys authorize sensitive ops like `vault operator generate-root` or rekey/recovery-rekey.

## Initialization

```bash
# Standard Shamir init: 5 shares, 3 to reconstruct
vault operator init -key-shares=5 -key-threshold=3

# Output (KEEP SECRET):
# Unseal Key 1: ...
# Unseal Key 2: ...
# ...
# Initial Root Token: hvs.AAAAAQ...

# JSON output for automation
vault operator init -format=json -key-shares=5 -key-threshold=3 > init.json
jq -r '.unseal_keys_b64[]' init.json
jq -r '.root_token' init.json

# Encrypt unseal keys with PGP keys (one per share)
vault operator init -key-shares=3 -key-threshold=2 \
  -pgp-keys="alice.asc,bob.asc,carol.asc" \
  -root-token-pgp-key="alice.asc"

# Auto-unseal: recovery shares replace unseal shares
vault operator init -recovery-shares=5 -recovery-threshold=3

# Stored shares + auto-unseal (Enterprise: store master key in KMS)
vault operator init -stored-shares=1 -recovery-shares=5 -recovery-threshold=3

# Check init status without initializing
vault operator init -status
# (exit 0 if uninit, 2 if init)
```

### Root Token Bootstrap

The initial root token has total privileges. Best practice: revoke it immediately after creating an admin policy and admin user/AppRole.

```bash
# After init, log in
export VAULT_TOKEN=hvs.AAAAAQ...
vault token lookup

# Create admin policy
vault policy write admin - <<EOF
path "*" { capabilities = ["create","read","update","delete","list","sudo"] }
EOF

# Enable userpass auth
vault auth enable userpass
vault write auth/userpass/users/admin \
  password=changeme token_policies=admin

# Verify, then revoke the bootstrap root
vault token revoke -self
```

### Recovery Shares (Auto-Unseal)

```bash
# After auto-unseal init, recovery key needed for these:
vault operator generate-root            # New root token; consumes recovery threshold
vault operator rekey -recovery-key      # Rekey recovery shares
vault operator seal-migration            # Switch seal type
```

## Sealing/Unsealing

### Manual Unseal (Shamir)

```bash
# Each operator runs in turn
vault operator unseal
# Prompts for key; running threshold count required

# Inline key (avoid in shell history)
vault operator unseal $UNSEAL_KEY_1
vault operator unseal $UNSEAL_KEY_2
vault operator unseal $UNSEAL_KEY_3

# Reset progress mid-flight
vault operator unseal -reset

# Migration (changing seal type)
vault operator unseal -migrate
```

### Sealing

```bash
# Seal an unsealed Vault (requires sudo capability)
vault operator seal

# Forced HA-aware step-down + seal
vault operator step-down

# Status
vault status
vault read sys/seal-status
```

### Auto-Unseal Flow

1. Vault starts; reads `seal "..."` stanza.
2. Storage contains encrypted master key.
3. Vault calls KMS to decrypt → master key in memory.
4. Master key decrypts data encryption key.
5. Vault unsealed automatically; recovery keys never needed for normal startup.

```bash
# Verify auto-unseal seal type
vault status | grep '^Type'
# Type     awskms

# Force a re-seal-then-restart for testing (Enterprise)
# Service-level: systemctl restart vault
```

### Seal Migration

```bash
# Migrate Shamir -> auto-unseal
# 1. Add new seal stanza AND keep old shamir
seal "awskms" {
  region     = "us-east-1"
  kms_key_id = "alias/vault-unseal"
  disabled   = "true"   # initially
}

# 2. Restart, then unseal with -migrate flag and Shamir keys
vault operator unseal -migrate $KEY1
vault operator unseal -migrate $KEY2
vault operator unseal -migrate $KEY3

# 3. Remove disabled = "true", restart; now using auto-unseal
# 4. Operator init recovery shares emerge
```

## Token Management

### Create

```bash
# Most-used flags
vault token create \
  -policy=app-read \
  -policy=audit-log \
  -ttl=24h \
  -explicit-max-ttl=72h \
  -display-name=ci-runner \
  -metadata=team=platform \
  -metadata=env=prod \
  -orphan \
  -no-default-policy \
  -renewable=true \
  -period=72h \
  -use-limit=10 \
  -wrap-ttl=5m \
  -entity-alias=alice
```

Flag breakdown:

```bash
-policy=NAME             # Attach policy (repeatable)
-ttl=DUR                 # Initial TTL (e.g. 1h, 24h, 72h)
-explicit-max-ttl=DUR    # Hard ceiling — token cannot exceed this
-period=DUR              # Periodic token: renews to this period; never expires unless revoked
-renewable=BOOL          # Default true
-orphan                  # No parent; not revoked when creator's token revoked (requires sudo)
-no-default-policy       # Skip the implicit "default" policy
-display-name=STR        # Friendly name in audit logs
-metadata=K=V            # Metadata, repeatable
-use-limit=N             # Token usable N times then auto-revoked (0 = unlimited)
-wrap-ttl=DUR            # Return wrapping token instead; unwrap to get real token
-entity-alias=STR        # Bind to identity entity alias
-role=NAME               # Use a token role (preset constraints)
-id=STR                  # Custom token id (rare; sudo)
-format=json             # Machine-readable output
```

### Lookup / Renew / Revoke

```bash
# Self lookup
vault token lookup
vault token lookup -self

# Look up another token (sudo on auth/token/lookup-accessor)
vault token lookup -accessor 0e2e...   # Use accessor (non-secret)
vault token lookup hvs.CAESI...        # Or the token itself (secret)

# Renew
vault token renew                                # self
vault token renew -increment=24h
vault token renew $TOKEN
vault token renew -accessor 0e2e...

# Revoke
vault token revoke                               # self
vault token revoke $TOKEN
vault token revoke -accessor 0e2e...
vault token revoke -self
vault token revoke -mode=orphan $TOKEN           # revoke but spare children
vault token revoke -mode=path auth/userpass/login/alice  # revoke by mount path

# Capabilities — what can THIS token do at PATH?
vault token capabilities secret/data/foo
vault token capabilities $TOKEN secret/data/foo
# read, list, update, create, delete, sudo, root, deny
```

### Token Tree (Parent/Child)

By default, child tokens are revoked when their parent is revoked. Orphan tokens have no parent (creating them requires sudo on `auth/token/create-orphan` or `-orphan` with sudo).

```bash
# Default child
vault token create                              # parent = current token

# Orphan: cannot be revoked via parent
vault token create -orphan

# Orphan via dedicated endpoint (sudo)
vault write -force auth/token/create-orphan
```

### Periodic vs Renewable

Periodic tokens have no max TTL — renewing resets TTL to the period. Useful for long-running services. Periodic creation requires sudo or a token role with `period`.

Renewable tokens have an `explicit_max_ttl` (or system max) — no amount of renewal extends past this point.

```bash
# Periodic (no expiration as long as renewed)
vault token create -period=72h -policy=app

# Renewable with hard ceiling
vault token create -ttl=1h -explicit-max-ttl=24h -policy=app
```

## Auth Methods Catalog

### List / Enable / Tune

```bash
vault auth list                            # show enabled mounts
vault auth list -detailed
vault auth enable userpass                 # mount at userpass/
vault auth enable -path=ldap-corp ldap     # custom mount path
vault auth disable userpass
vault auth tune -default-lease-ttl=8h userpass/
vault auth tune -listing-visibility=unauth userpass/
```

### Methods

```bash
# token        — built-in, always present at /auth/token
# userpass     — username/password, bcrypt
vault auth enable userpass
vault write auth/userpass/users/alice password=s3cret token_policies=app

# ldap         — LDAP/AD bind + group mapping
vault auth enable ldap
vault write auth/ldap/config \
  url="ldaps://ldap.example.com" \
  userdn="ou=Users,dc=example,dc=com" \
  groupdn="ou=Groups,dc=example,dc=com" \
  binddn="cn=vault,ou=Service,dc=example,dc=com" \
  bindpass='s3cret' \
  userattr=uid

# oidc         — OpenID Connect (browser flow)
vault auth enable oidc

# jwt          — Signed JWT (CI, K8s, GitHub Actions)
vault auth enable jwt

# github       — GitHub team membership
vault auth enable github
vault write auth/github/config organization=acmecorp
vault write auth/github/map/teams/platform value=admin

# kubernetes   — In-cluster ServiceAccount JWT
vault auth enable kubernetes

# aws          — IAM signed sts:GetCallerIdentity OR EC2 PKCS7
vault auth enable aws

# gcp          — GCE/IAM signed JWT
vault auth enable gcp

# azure        — Managed Identity / VM signed token
vault auth enable azure

# approle      — RoleID + SecretID (machine auth)
vault auth enable approle

# cert         — Client TLS certificate (mTLS)
vault auth enable cert

# tls-cert     — alias of cert

# radius       — RADIUS server
vault auth enable radius
vault write auth/radius/config host=1.2.3.4 secret='shared'
```

## AppRole Workflow

Machine-to-machine auth: a Role represents an application; RoleID identifies it; SecretID is the password (often delivered out-of-band, e.g. via response wrapping).

### Enable

```bash
vault auth enable approle

# Tune TTL ceiling
vault auth tune -max-lease-ttl=24h approle/
```

### Write Role

```bash
vault write auth/approle/role/ci-runner \
  token_policies="ci-read,ci-write" \
  token_ttl=1h \
  token_max_ttl=4h \
  token_num_uses=0 \
  token_period=0 \
  bind_secret_id=true \
  secret_id_ttl=24h \
  secret_id_num_uses=10 \
  token_bound_cidrs="10.0.0.0/8" \
  secret_id_bound_cidrs="10.0.0.0/8" \
  enable_local_secret_ids=false
```

Useful flags:

```bash
token_policies=...        # Comma-sep policies attached to issued tokens
token_ttl=DUR             # Initial TTL of issued tokens
token_max_ttl=DUR         # Max TTL ceiling
token_period=DUR          # Make tokens periodic
token_num_uses=N          # Hard use limit
bind_secret_id=BOOL       # Require SecretID; false = RoleID alone (CIDR-bound)
secret_id_ttl=DUR         # SecretID expiry
secret_id_num_uses=N      # SecretID can be exchanged N times
secret_id_bound_cidrs=CIDR # SecretID only usable from these networks
token_bound_cidrs=CIDR    # Issued tokens only usable from these networks
```

### Read RoleID

```bash
# RoleID is non-secret; can be baked into CI config
vault read auth/approle/role/ci-runner/role-id
# role_id   abcd1234-...
```

### Generate SecretID

```bash
# Random
vault write -f auth/approle/role/ci-runner/secret-id

# Custom (rare)
vault write auth/approle/role/ci-runner/custom-secret-id \
  secret_id="my-preset-secret"

# Wrapped (one-shot delivery to consumer)
vault write -wrap-ttl=60s -f auth/approle/role/ci-runner/secret-id
# wrapping_token: hvs.CAESI...
# Consumer unwraps: VAULT_TOKEN=$WRAP vault unwrap
```

### Login Flow

```bash
# Exchange RoleID + SecretID for a token
vault write auth/approle/login \
  role_id=abcd1234-... \
  secret_id=ef567890-...

# Output:
# token         hvs.CAESI...
# token_accessor ...
# token_policies [default ci-read ci-write]
# token_duration 1h

# Capture and use
TOKEN=$(vault write -format=json auth/approle/login \
  role_id=$RID secret_id=$SID | jq -r .auth.client_token)
export VAULT_TOKEN=$TOKEN
```

## Kubernetes Auth

In-cluster pods present their ServiceAccount JWT; Vault validates it against the cluster API server.

### Enable + Configure

```bash
vault auth enable kubernetes

# Inside the cluster (Vault running as a pod)
vault write auth/kubernetes/config \
  kubernetes_host="https://kubernetes.default.svc.cluster.local" \
  kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt \
  token_reviewer_jwt=@/var/run/secrets/kubernetes.io/serviceaccount/token \
  issuer="https://kubernetes.default.svc.cluster.local" \
  disable_iss_validation=false

# Outside cluster: pass kubeconfig-derived host + a long-lived reviewer JWT
vault write auth/kubernetes/config \
  kubernetes_host="https://k8s.example.com:6443" \
  kubernetes_ca_cert=@/etc/k8s/ca.pem \
  token_reviewer_jwt="$(cat reviewer.jwt)"
```

### Bind Role

```bash
vault write auth/kubernetes/role/web-app \
  bound_service_account_names=web-app \
  bound_service_account_namespaces=production \
  token_policies=web-app \
  token_ttl=1h \
  audience=vault \
  alias_name_source=serviceaccount_uid

# Wildcards
bound_service_account_names="*"
bound_service_account_namespaces="team-*"
```

### Pod Login

```bash
JWT=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
vault write auth/kubernetes/login \
  role=web-app jwt="$JWT"
```

## OIDC/JWT Auth

OIDC: browser-mediated user login (Okta, Auth0, Google, GitHub OIDC).
JWT: signed token validated against JWKS or static key (CI runners, GitHub Actions).

### OIDC Configure

```bash
vault auth enable oidc

vault write auth/oidc/config \
  oidc_discovery_url="https://accounts.example.com" \
  oidc_client_id="vault-cli" \
  oidc_client_secret="..." \
  default_role="reader" \
  oidc_response_mode="form_post" \
  oidc_response_types="code"
```

### OIDC Role

```bash
vault write auth/oidc/role/reader \
  user_claim="email" \
  bound_audiences="vault-cli" \
  allowed_redirect_uris="http://localhost:8250/oidc/callback" \
  allowed_redirect_uris="https://vault.example.com/ui/vault/auth/oidc/oidc/callback" \
  groups_claim="groups" \
  oidc_scopes="openid,profile,email,groups" \
  token_policies="reader" \
  token_ttl=1h \
  verbose_oidc_logging=true
```

### OIDC Login

```bash
# CLI opens a browser, listens on localhost:8250
vault login -method=oidc role=reader
# Or: VAULT_OIDC_CALLBACK_LISTENER=127.0.0.1:18250
```

### JWT (e.g. GitHub Actions)

```bash
vault auth enable -path=jwt jwt

vault write auth/jwt/config \
  oidc_discovery_url="https://token.actions.githubusercontent.com" \
  bound_issuer="https://token.actions.githubusercontent.com"

vault write auth/jwt/role/gha-deploy \
  role_type="jwt" \
  user_claim="actor" \
  bound_claims_type="glob" \
  bound_claims='{"repository":"acmecorp/*","ref":"refs/heads/main"}' \
  bound_audiences="https://github.com/acmecorp" \
  token_policies="ci-deploy" \
  token_ttl=15m

# In CI:
JWT=$(curl -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
  "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=https://github.com/acmecorp" \
  | jq -r .value)
vault write auth/jwt/login role=gha-deploy jwt="$JWT"
```

## AWS Auth

### IAM (signed sts:GetCallerIdentity)

Caller signs an STS GetCallerIdentity request; Vault verifies the signature against AWS.

```bash
vault auth enable aws
vault write auth/aws/config/client \
  access_key=AKIA... \
  secret_key=...

vault write auth/aws/role/ec2-app \
  auth_type=iam \
  bound_iam_principal_arn="arn:aws:iam::123456789012:role/EC2App" \
  policies=app \
  token_ttl=1h \
  token_max_ttl=4h \
  resolve_aws_unique_ids=true

# Login from EC2 / Lambda / ECS task with that role
vault login -method=aws role=ec2-app

# Manual
SIGN=$(aws sts get-caller-identity --query "..." ...)
vault write auth/aws/login role=ec2-app \
  iam_http_request_method=POST \
  iam_request_url=$URL \
  iam_request_body=$BODY \
  iam_request_headers=$HEADERS
```

### EC2 (PKCS7)

EC2 instance presents its instance identity document signed by AWS.

```bash
vault write auth/aws/role/legacy-ec2 \
  auth_type=ec2 \
  bound_account_id=123456789012 \
  bound_ami_id=ami-abc123 \
  bound_vpc_id=vpc-12345 \
  bound_subnet_id=subnet-12345 \
  bound_iam_role_arn="arn:aws:iam::123456789012:role/EC2App" \
  bound_iam_instance_profile_arn="arn:aws:iam::123456789012:instance-profile/EC2App" \
  bound_region=us-east-1 \
  policies=legacy
```

```bash
PKCS7=$(curl -s http://169.254.169.254/latest/dynamic/instance-identity/pkcs7 | tr -d '\n')
NONCE=$(uuidgen)
vault write auth/aws/login role=legacy-ec2 pkcs7="$PKCS7" nonce="$NONCE"
```

## Secrets: kv-v2

Versioned KV store. Default mount in dev mode is `secret/` (kv-v2). Path semantics differ from kv-v1: data lives under `<mount>/data/<path>`.

### Enable

```bash
vault secrets enable -version=2 -path=secret kv
# or rename kv to kv-v2 explicitly
vault secrets enable -path=kv kv-v2

vault secrets tune -default-lease-ttl=0s secret/
```

### Put / Get

```bash
# Put (creates new version)
vault kv put secret/db/postgres \
  username=app \
  password='s3cret!'

# Put from JSON file
vault kv put secret/db/postgres @data.json

# Put from stdin
echo '{"username":"app","password":"x"}' | vault kv put secret/db/postgres -

# Get latest
vault kv get secret/db/postgres

# Get specific field
vault kv get -field=password secret/db/postgres

# Get specific version
vault kv get -version=3 secret/db/postgres

# JSON
vault kv get -format=json secret/db/postgres | jq -r .data.data.password
```

### Versioning / Metadata

```bash
# Configure
vault kv metadata put -max-versions=10 -delete-version-after=720h \
  secret/db/postgres
vault kv metadata put -cas-required=true secret/db/postgres
vault kv metadata put -custom-metadata=team=platform secret/db/postgres

# Inspect
vault kv metadata get secret/db/postgres

# CAS (compare-and-swap) write — required if cas-required=true
vault kv put -cas=3 secret/db/postgres password=new

# Soft-delete a version (recoverable)
vault kv delete secret/db/postgres
vault kv delete -versions=2,3 secret/db/postgres

# Undelete
vault kv undelete -versions=2,3 secret/db/postgres

# Permanent destroy
vault kv destroy -versions=2 secret/db/postgres

# Delete all metadata + data (irreversible)
vault kv metadata delete secret/db/postgres
```

### Path Semantics

```bash
# CLI            REAL API PATH
# secret/db/x -> secret/data/db/x         (data ops)
#             -> secret/metadata/db/x     (metadata ops)
#             -> secret/delete/db/x       (soft delete)
#             -> secret/undelete/db/x
#             -> secret/destroy/db/x
#             -> secret/subkeys/db/x      (1.13+)

# In policies, ALWAYS use the API path (data/, metadata/...)
path "secret/data/db/*"     { capabilities = ["read"] }
path "secret/metadata/db/*" { capabilities = ["list","read"] }
```

## Secrets: kv-v1

Legacy non-versioned KV. Simpler — no `data/` indirection.

```bash
vault secrets enable -version=1 -path=kv1 kv

vault write kv1/db/postgres username=app password=s3cret
vault read  kv1/db/postgres
vault delete kv1/db/postgres
vault list  kv1/

# Policy path (no data/ prefix)
path "kv1/db/*" { capabilities = ["read"] }
```

## Secrets: transit

Encryption-as-a-service. Vault never stores plaintext; it encrypts/decrypts on demand using a named key.

### Enable + Create Keys

```bash
vault secrets enable transit

# Default: aes256-gcm96
vault write -f transit/keys/orders

# Named key types
vault write transit/keys/orders type=aes256-gcm96
vault write transit/keys/email  type=chacha20-poly1305
vault write transit/keys/sigs   type=ed25519
vault write transit/keys/ec     type=ecdsa-p256
vault write transit/keys/rsa    type=rsa-2048
vault write transit/keys/rsa4k  type=rsa-4096

# Convergent (deterministic) — same plaintext always yields same ciphertext
vault write transit/keys/dedup \
  type=aes256-gcm96 \
  convergent_encryption=true \
  derived=true

# Read key info
vault read transit/keys/orders
```

### Encrypt / Decrypt

```bash
# Plaintext must be base64-encoded
PT=$(base64 <<< "card-1234-5678-9012-3456")
CIPHER=$(vault write -field=ciphertext transit/encrypt/orders \
  plaintext=$PT)
echo $CIPHER
# vault:v1:abcd1234...

# Decrypt
vault write -field=plaintext transit/decrypt/orders ciphertext=$CIPHER \
  | base64 -d

# With context (derived/convergent)
CTX=$(base64 <<< "user-42")
vault write transit/encrypt/dedup plaintext=$PT context=$CTX

# Batch
vault write transit/encrypt/orders batch_input=- <<EOF
{"batch_input":[{"plaintext":"$PT"},{"plaintext":"$PT2"}]}
EOF
```

### Rotation

```bash
# Add a new version (new keys.N entry)
vault write -f transit/keys/orders/rotate

# Rewrap old ciphertext to current version
vault write transit/rewrap/orders ciphertext=$CIPHER

# Constrain decrypt to recent versions only
vault write transit/keys/orders/config \
  min_decryption_version=3 \
  min_encryption_version=4 \
  deletion_allowed=false
```

### Sign / Verify

```bash
# Hash + sign
vault write -field=signature transit/sign/sigs/sha2-256 \
  input=$(base64 <<< "msg")

vault write transit/verify/sigs/sha2-256 \
  input=$(base64 <<< "msg") signature=vault:v1:...

# RSA-PSS
vault write transit/sign/rsa/sha2-256 \
  input=$PT signature_algorithm=pss

# Sign prehashed
vault write transit/sign/sigs \
  input=$HASH prehashed=true \
  hash_algorithm=sha2-512
```

### HMAC

```bash
vault write -field=hmac transit/hmac/orders/sha2-256 \
  input=$(base64 <<< "msg")
# vault:v1:abcd1234...

vault write transit/verify/orders/sha2-256 \
  input=$(base64 <<< "msg") hmac=vault:v1:...
```

## Secrets: database

Dynamic credentials — Vault creates a short-lived database user on demand and revokes it when the lease expires.

### Enable

```bash
vault secrets enable database
vault secrets tune -max-lease-ttl=24h database/
```

### PostgreSQL

```bash
# Connection
vault write database/config/pg \
  plugin_name=postgresql-database-plugin \
  connection_url="postgresql://{{username}}:{{password}}@pg:5432/app?sslmode=require" \
  allowed_roles="readonly,readwrite" \
  username="vault_admin" \
  password="..." \
  password_authentication="scram-sha-256" \
  max_open_connections=5 \
  max_connection_lifetime=300s

# Role (dynamic)
vault write database/roles/readonly \
  db_name=pg \
  creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; GRANT SELECT ON ALL TABLES IN SCHEMA public TO \"{{name}}\";" \
  revocation_statements="REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM \"{{name}}\"; DROP ROLE \"{{name}}\";" \
  default_ttl=1h \
  max_ttl=24h

# Read creds (creates a user)
vault read database/creds/readonly
# username   v-token-readonly-AbCdEf-...
# password   ...
# lease_id   database/creds/readonly/...

# Static role: Vault rotates the password of an existing user on a schedule
vault write database/static-roles/app \
  db_name=pg \
  username=app_static \
  rotation_period=24h \
  rotation_statements="ALTER USER \"{{name}}\" WITH PASSWORD '{{password}}';"

vault read database/static-creds/app
```

### MySQL

```bash
vault write database/config/mysql \
  plugin_name=mysql-database-plugin \
  connection_url="{{username}}:{{password}}@tcp(mysql:3306)/" \
  allowed_roles="ro,rw" \
  username=vaultadmin password=...

vault write database/roles/ro \
  db_name=mysql \
  creation_statements="CREATE USER '{{name}}'@'%' IDENTIFIED BY '{{password}}'; GRANT SELECT ON app.* TO '{{name}}'@'%';" \
  default_ttl=1h max_ttl=24h
```

### MSSQL

```bash
vault write database/config/mssql \
  plugin_name=mssql-database-plugin \
  connection_url='sqlserver://{{username}}:{{password}}@mssql:1433' \
  allowed_roles=app username=vaultadmin password=...

vault write database/roles/app \
  db_name=mssql \
  creation_statements="CREATE LOGIN [{{name}}] WITH PASSWORD = '{{password}}'; CREATE USER [{{name}}] FOR LOGIN [{{name}}]; GRANT SELECT TO [{{name}}];" \
  default_ttl=1h
```

### MongoDB

```bash
vault write database/config/mongo \
  plugin_name=mongodb-database-plugin \
  connection_url="mongodb://{{username}}:{{password}}@mongo:27017/admin?ssl=true" \
  allowed_roles=app username=vaultadmin password=...

vault write database/roles/app \
  db_name=mongo \
  creation_statements='{ "db": "app", "roles": [{"role":"readWrite","db":"app"}] }' \
  default_ttl=1h
```

### Cassandra

```bash
vault write database/config/cassandra \
  plugin_name=cassandra-database-plugin \
  hosts=cassandra \
  port=9042 \
  username=cassandra password=cassandra \
  allowed_roles=app

vault write database/roles/app \
  db_name=cassandra \
  creation_statements="CREATE USER '{{username}}' WITH PASSWORD '{{password}}' NOSUPERUSER;" \
  default_ttl=1h
```

### Redis

```bash
vault write database/config/redis \
  plugin_name=redis-database-plugin \
  host=redis port=6379 \
  username=default password=... \
  tls=true insecure_tls=false ca_cert=@ca.pem \
  allowed_roles=app

vault write database/roles/app \
  db_name=redis \
  creation_statements='["+@read", "~app:*"]' \
  default_ttl=1h
```

### Snowflake / Oracle / Elasticsearch

```bash
# Snowflake
vault write database/config/snow \
  plugin_name=snowflake-database-plugin \
  connection_url="{{username}}:{{password}}@acme.snowflakecomputing.com/db/schema?warehouse=wh" \
  allowed_roles=etl \
  username=VAULTADMIN password=...

# Oracle (custom plugin)
vault write database/config/ora \
  plugin_name=oracle-database-plugin \
  connection_url="{{username}}/{{password}}@//db:1521/XE" \
  allowed_roles=app username=VAULT password=...

# Elasticsearch
vault write database/config/es \
  plugin_name=elasticsearch-database-plugin \
  url=https://es:9200 \
  username=elastic password=... \
  ca_cert=@ca.pem \
  allowed_roles=app

vault write database/roles/app \
  db_name=es \
  creation_statements='{"elasticsearch_roles":["read_app"]}' \
  default_ttl=1h
```

### Lease Reissue

```bash
vault read database/creds/readonly      # new user each call
vault lease renew database/creds/readonly/AbCdEf
vault lease revoke database/creds/readonly/AbCdEf
vault lease revoke -prefix database/creds/readonly
```

## Secrets: AWS

Dynamic IAM creds.

### Root Config

```bash
vault secrets enable aws
vault write aws/config/root \
  access_key=AKIA... \
  secret_key=... \
  region=us-east-1 \
  iam_endpoint=https://iam.amazonaws.com \
  sts_endpoint=https://sts.us-east-1.amazonaws.com \
  max_retries=3

vault write aws/config/lease \
  lease=1h lease_max=24h
```

### Roles

```bash
# iam_user (creates real IAM user; slow, hard limit)
vault write aws/roles/s3-readonly \
  credential_type=iam_user \
  policy_arns="arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"

# Inline policy
vault write aws/roles/s3-list \
  credential_type=iam_user \
  policy_document=-<<EOF
{ "Version":"2012-10-17","Statement":[{
  "Effect":"Allow","Action":["s3:ListBucket"],"Resource":"*" }] }
EOF

# sts_federation_token (faster; up to 36h)
vault write aws/roles/fed \
  credential_type=federation_token \
  policy_arns="arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"

# sts_assume_role (cross-account)
vault write aws/roles/assume \
  credential_type=assumed_role \
  role_arns="arn:aws:iam::123456789012:role/CrossAccount"
```

### Read

```bash
vault read aws/creds/s3-readonly
# access_key   AKIA...
# secret_key   ...
# security_token  (sts roles only)
# lease_id ...

vault read aws/sts/fed ttl=900
```

## Secrets: PKI

Vault as a CA.

### Enable + Configure

```bash
vault secrets enable pki
vault secrets tune -max-lease-ttl=87600h pki/

# Generate self-signed root (or import via /config/ca)
vault write pki/root/generate/internal \
  common_name="acme Root CA" \
  ttl=87600h \
  key_type=rsa key_bits=4096 \
  organization=Acme \
  country=US

# URLs in issued certs
vault write pki/config/urls \
  issuing_certificates="https://vault.example.com/v1/pki/ca" \
  crl_distribution_points="https://vault.example.com/v1/pki/crl"
```

### Role

```bash
vault write pki/roles/web \
  allowed_domains="example.com,example.org" \
  allow_subdomains=true \
  allow_bare_domains=false \
  allow_glob_domains=false \
  allow_localhost=false \
  allow_ip_sans=true \
  enforce_hostnames=true \
  client_flag=false server_flag=true \
  key_type=rsa key_bits=2048 \
  ttl=720h max_ttl=2160h \
  generate_lease=false \
  no_store=false
```

### Issue / Revoke

```bash
# Issue: Vault generates the keypair AND returns private key
vault write pki/issue/web \
  common_name=app.example.com \
  alt_names=app2.example.com \
  ip_sans=10.1.2.3 \
  ttl=72h
# certificate, private_key, ca_chain, serial_number

# Sign existing CSR (private key never leaves client)
vault write pki/sign/web \
  csr=@app.csr \
  common_name=app.example.com ttl=72h

# Revoke
vault write pki/revoke serial_number="2a:b1:..."

# CRL
curl -s $VAULT_ADDR/v1/pki/crl                       # DER
curl -s $VAULT_ADDR/v1/pki/crl/pem                   # PEM
vault write pki/crl/rotate                            # force rotate
```

### Intermediate CA

```bash
# At intermediate Vault: generate CSR
vault secrets enable -path=pki_int pki
vault secrets tune -max-lease-ttl=43800h pki_int/

vault write -format=json pki_int/intermediate/generate/internal \
  common_name="acme Intermediate CA" ttl=43800h \
  | jq -r .data.csr > int.csr

# At root: sign CSR
vault write -format=json pki/root/sign-intermediate \
  csr=@int.csr format=pem_bundle ttl=43800h \
  | jq -r .data.certificate > int.crt

# Back at intermediate: import signed cert
vault write pki_int/intermediate/set-signed certificate=@int.crt
```

## Secrets: SSH

### CA Mode (signed certificates)

```bash
vault secrets enable ssh
vault write ssh/config/ca generate_signing_key=true
vault read -field=public_key ssh/config/ca > vault-ssh-ca.pub

# Distribute vault-ssh-ca.pub to servers' sshd:
# /etc/ssh/sshd_config:
#   TrustedUserCAKeys /etc/ssh/vault-ssh-ca.pub

# Role for users
vault write ssh/roles/admin \
  key_type=ca \
  algorithm_signer=rsa-sha2-256 \
  allow_user_certificates=true \
  allowed_users="ubuntu,ec2-user,root" \
  default_user="ubuntu" \
  default_extensions='{"permit-pty":""}' \
  allowed_extensions="permit-pty,permit-port-forwarding" \
  ttl=4h max_ttl=24h

# Sign user's pubkey
vault write -field=signed_key ssh/sign/admin \
  public_key=@$HOME/.ssh/id_ed25519.pub \
  valid_principals=ubuntu \
  > ~/.ssh/id_ed25519-cert.pub

ssh -i ~/.ssh/id_ed25519 ubuntu@host
```

### OTP Mode

```bash
vault secrets enable -path=ssh-otp ssh
vault write ssh-otp/roles/otp \
  key_type=otp \
  default_user=ubuntu \
  cidr_list=10.0.0.0/8 \
  port=22

vault write ssh-otp/creds/otp ip=10.0.1.5
# key  abc-123-...     (OTP password)
# Use vault-ssh-helper PAM module on host to consume
```

### SSH Host Certs

```bash
vault write ssh/roles/host \
  key_type=ca \
  allow_host_certificates=true \
  allowed_domains="example.com" \
  allow_subdomains=true \
  ttl=8760h

vault write ssh/sign/host \
  cert_type=host \
  public_key=@/etc/ssh/ssh_host_ed25519_key.pub \
  valid_principals=host01.example.com

# /etc/ssh/sshd_config:
#   HostCertificate /etc/ssh/ssh_host_ed25519_key-cert.pub
#   HostKey         /etc/ssh/ssh_host_ed25519_key
```

## Secrets: TOTP

Generate or validate Google-Authenticator-style codes.

```bash
vault secrets enable totp

# Generated key — Vault makes the secret + URL
vault write totp/keys/alice \
  generate=true \
  issuer=Acme \
  account_name=alice@example.com \
  qr_size=200
# url: otpauth://totp/Acme:alice@...

# Imported (from existing secret)
vault write totp/keys/bob \
  url="otpauth://totp/Acme:bob?secret=JBSWY3DPEHPK3PXP&issuer=Acme"

# Read current code
vault read totp/code/alice
# code  123456

# Validate user-supplied code
vault write totp/code/alice code=123456
# valid  true
```

## Secrets: Transform (Enterprise)

Format-Preserving Encryption (FPE), tokenization, masking.

```bash
vault secrets enable transform

# FPE on a credit card pattern
vault write transform/role/payments transformations=ccn
vault write transform/transformations/fpe/ccn \
  template=builtin/creditcardnumber \
  tweak_source=internal \
  allowed_roles=payments

vault write transform/encode/payments value=4111-1111-1111-1111
# encoded_value 4444-5555-6666-7777    (still 19 chars)

vault write transform/decode/payments value=4444-5555-6666-7777

# Tokenization (random surrogate, irreversible without Vault)
vault write transform/transformations/tokenization/email \
  allowed_roles=marketing \
  max_ttl=720h

# Masking (one-way)
vault write transform/transformations/masking/ssn \
  template=builtin/socialsecuritynumber \
  masking_character=*
```

## Secrets: Kubernetes

Generate ephemeral K8s ServiceAccount tokens scoped to a namespace.

```bash
vault secrets enable kubernetes
vault write kubernetes/config \
  kubernetes_host="https://k8s.example.com:6443" \
  kubernetes_ca_cert=@/etc/k8s/ca.pem \
  service_account_jwt=@/etc/k8s/admin.jwt

vault write kubernetes/roles/app \
  allowed_kubernetes_namespaces="team-a,team-b" \
  service_account_name="app-sa" \
  token_default_ttl=1h \
  token_max_ttl=4h \
  token_default_audiences="vault" \
  generated_role_rules=- <<EOF
rules:
- apiGroups: [""]
  resources: ["pods","services"]
  verbs: ["get","list","watch"]
EOF

# Reads ephemeral token
vault write kubernetes/creds/app \
  kubernetes_namespace=team-a \
  cluster_role_binding=false \
  ttl=1h
```

## Policies

HCL syntax. Match by exact path or `*` (wildcard) or `+` (single-segment).

### Capabilities

```bash
# create   POST  to nonexistent path
# read     GET
# update   PUT/POST to existing path
# delete   DELETE
# list     LIST
# sudo     bypass root-protected endpoints (e.g. sys/raw)
# deny     overrides everything (highest precedence)
# root     reserved for the root token; cannot be granted
```

### Examples

```bash
# Read-only KV access
path "secret/data/team-a/*" {
  capabilities = ["read","list"]
}
path "secret/metadata/team-a/*" {
  capabilities = ["list","read"]
}

# Self-renew + lookup
path "auth/token/renew-self"  { capabilities = ["update"] }
path "auth/token/lookup-self" { capabilities = ["read"] }

# Templated path: each user sees only their folder
path "secret/data/users/{{identity.entity.aliases.auth_userpass_abc.name}}/*" {
  capabilities = ["create","read","update","delete","list"]
}

path "secret/data/teams/{{identity.entity.metadata.team}}/*" {
  capabilities = ["read","list"]
}

# Deny override
path "secret/data/admin/*"  { capabilities = ["read"] }
path "secret/data/admin/superuser/*" { capabilities = ["deny"] }

# Allowed/denied/required parameters
path "transit/encrypt/orders" {
  capabilities = ["update"]
  allowed_parameters = {
    "context"   = []
    "plaintext" = []
  }
  denied_parameters = {
    "key_version" = []
  }
  required_parameters = ["plaintext"]
}

# Wrapping TTL bounds
path "auth/approle/role/+/secret-id" {
  capabilities = ["update"]
  min_wrapping_ttl = "60s"
  max_wrapping_ttl = "10m"
}

# kv-v1: no data/ prefix
path "kv1/team-a/*" { capabilities = ["read","list"] }
```

### Manage Policies

```bash
vault policy list
vault policy read default
vault policy write app-read app-read.hcl
vault policy write app-read - <<EOF
path "secret/data/app/*" { capabilities = ["read"] }
EOF
vault policy delete app-read

# Format check
vault policy fmt app-read.hcl
```

### data/ vs metadata/ in kv-v2

Forgetting `data/` is the most common policy bug. The CLI hides it; the policy must include it explicitly.

```bash
# WRONG (will fail with permission denied)
path "secret/team-a/*" { capabilities = ["read"] }

# RIGHT
path "secret/data/team-a/*"     { capabilities = ["read"] }
path "secret/metadata/team-a/*" { capabilities = ["list","read"] }
```

## Identity

Entities, aliases, and groups unify identities across multiple auth methods.

### Entity / Alias

```bash
# Create entity
vault write identity/entity \
  name=alice \
  policies=eng \
  metadata=team=platform \
  metadata=role=lead

ENT=$(vault read -field=id identity/entity/name/alice)

# Find auth mount accessor
ACCESSOR=$(vault auth list -format=json | jq -r '.["userpass/"].accessor')

# Bind alias
vault write identity/entity-alias \
  name=alice \
  canonical_id=$ENT \
  mount_accessor=$ACCESSOR

# Now alice's userpass/ldap/oidc identity all map to the same entity
vault read identity/entity/name/alice
```

### Groups

```bash
# Internal group: explicit membership
vault write identity/group \
  name=platform-leads \
  policies=admin \
  member_entity_ids=$ENT_ALICE,$ENT_BOB

# External group: pulled from auth method (LDAP groups, OIDC claim, etc.)
vault write identity/group \
  name=eng \
  type=external \
  policies=eng

GID=$(vault read -field=id identity/group/name/eng)

# Tie to LDAP group "engineering"
vault write identity/group-alias \
  name=engineering \
  mount_accessor=$LDAP_ACCESSOR \
  canonical_id=$GID
```

### Lookup

```bash
vault list identity/entity/id
vault read identity/entity/id/$ENT
vault read identity/entity/name/alice

vault list identity/group/name
vault read identity/group/name/eng
```

## Leases

Every dynamic secret + token has a lease. Vault revokes when the lease expires.

```bash
# Lookup
vault lease lookup database/creds/readonly/AbCdEf

# List active leases under a prefix
vault list sys/leases/lookup/database/creds/readonly

# Renew
vault lease renew database/creds/readonly/AbCdEf
vault lease renew -increment=2h database/creds/readonly/AbCdEf

# Revoke
vault lease revoke database/creds/readonly/AbCdEf
vault lease revoke -prefix database/creds/readonly
vault lease revoke -force -prefix database/   # ignore errors

# Tunables (per-mount)
vault secrets tune \
  -default-lease-ttl=1h \
  -max-lease-ttl=24h \
  database/

# Global ceilings (HCL server config)
default_lease_ttl = "768h"   # 32 days
max_lease_ttl     = "8760h"  # 1 year
```

The mount's `max_lease_ttl` truncates anything longer; the system `max_lease_ttl` truncates further. The smallest applies.

## Auto-Unseal

Server config snippets — only ONE seal stanza active (plus a disabled one during migration).

```bash
# AWS KMS
seal "awskms" {
  region     = "us-east-1"
  kms_key_id = "alias/vault-unseal"
  endpoint   = ""           # custom endpoint (LocalStack etc.)
}

# Azure Key Vault
seal "azurekeyvault" {
  tenant_id      = "..."
  client_id      = "..."   # or use MSI
  client_secret  = "..."
  vault_name     = "vault-unseal"
  key_name       = "unsealkey"
  environment    = "AzurePublicCloud"
}

# GCP KMS
seal "gcpckms" {
  credentials = "/etc/vault.d/gcp.json"
  project     = "my-project"
  region      = "global"
  key_ring    = "vault"
  crypto_key  = "vault-unseal"
}

# OCI KMS
seal "ocikms" {
  key_id              = "ocid1.key.oc1..."
  crypto_endpoint     = "https://...-crypto.kms.us-ashburn-1.oraclecloud.com"
  management_endpoint = "https://...-management.kms.us-ashburn-1.oraclecloud.com"
  auth_type_api_key   = "true"
}

# Transit (chained Vaults)
seal "transit" {
  address         = "https://vault-master:8200"
  token           = "..."        # or use VAULT_TOKEN env
  disable_renewal = "false"
  key_name        = "autounseal"
  mount_path      = "transit/"
  namespace       = "ns1/"
  tls_ca_cert     = "/etc/vault.d/ca.pem"
}

# HSM PKCS#11 (Enterprise)
seal "pkcs11" {
  lib            = "/usr/safenet/lunaclient/lib/libCryptoki2_64.so"
  slot           = "0"
  pin            = "..."
  key_label      = "vault-hsm-key"
  hmac_key_label = "vault-hsm-hmac"
  generate_key   = "true"
  mechanism      = "0x1085"      # CKM_AES_GCM
}
```

## Replication (Enterprise)

DR (disaster recovery): warm standby; takes over on failover. Cannot serve reads in normal operation. Pre-revokes all client tokens on promotion.

Performance (PR): horizontally scales reads across regions; secondaries serve API except for some write paths (which forward to primary).

### Activate Primary

```bash
# DR primary
vault write -f sys/replication/dr/primary/enable

# Performance primary
vault write -f sys/replication/performance/primary/enable

# Generate secondary token
vault write sys/replication/dr/primary/secondary-token \
  id=dr-east ttl=10m
# token: hvs.CAESI...

vault write sys/replication/performance/primary/secondary-token \
  id=perf-eu ttl=10m
```

### Activate Secondary

```bash
# DR secondary (note the seal config must be IDENTICAL on both)
vault write sys/replication/dr/secondary/enable \
  token=$DR_TOKEN \
  primary_api_addr=https://vault-east:8200 \
  ca_file=/etc/vault.d/ca.pem

# Performance secondary
vault write sys/replication/performance/secondary/enable \
  token=$PERF_TOKEN \
  primary_api_addr=https://vault-east:8200
```

### Filtering

```bash
# Allow / deny specific mounts on a perf secondary
vault write sys/replication/performance/primary/paths-filter/perf-eu \
  mode=deny \
  paths="us-only/,sensitive/"

# Or allow-only
vault write sys/replication/performance/primary/paths-filter/perf-eu \
  mode=allow \
  paths="public/,kv/"
```

### Promotion / Demotion

```bash
# DR: failover
vault write sys/replication/dr/secondary/promote dr_operation_token=$OP_TOKEN

# Generate dr_operation_token (multi-shard recovery key consent)
vault write sys/replication/dr/secondary/generate-operation-token/attempt
vault write sys/replication/dr/secondary/generate-operation-token/update \
  key=$RECOVERY_SHARE nonce=...

# Demote primary -> secondary
vault write -f sys/replication/dr/primary/demote
```

## Namespaces (Enterprise)

Tenant isolation: each namespace has its own auth methods, secret engines, policies, identities. Nested.

```bash
vault namespace create team-a
vault namespace create -namespace=team-a engineering

vault namespace list
# team-a/
# team-a/engineering/

# Operate within a namespace
export VAULT_NAMESPACE=team-a
vault secrets enable kv
vault kv put secret/foo bar=baz

# Or per-call
vault kv put -namespace=team-a secret/foo bar=baz

# HTTP header
curl -H "X-Vault-Namespace: team-a" $VAULT_ADDR/v1/secret/data/foo

vault namespace delete team-a/engineering
```

## Audit Devices

Every authenticated request is logged. Vault refuses to operate if all enabled audit devices fail (fail-closed).

### File

```bash
vault audit enable file file_path=/var/log/vault/audit.log
vault audit enable -path=stdout-audit file file_path=stdout
vault audit enable file \
  file_path=/var/log/vault/audit.log \
  log_raw=false \
  hmac_accessor=true \
  mode=0600 \
  format=json \
  prefix="" \
  fallback=false

vault audit list -detailed
vault audit disable file/
```

### Syslog

```bash
vault audit enable syslog tag=vault facility=AUTH
```

### Socket

```bash
vault audit enable socket \
  address=10.0.1.5:9090 \
  socket_type=tcp \
  format=json
```

### Format / HMAC

Each event is a JSON envelope. Secret values are HMAC-SHA256 hashed using a per-Vault audit key. Identical secrets across events have identical hashes — so you can detect reuse without leaking plaintext.

```bash
# Sample audit event:
# {"time":"2026-04-25T...","type":"response",
#  "auth":{"client_token":"hmac-sha256:...","accessor":"...","policies":[]},
#  "request":{"id":"...","operation":"read","path":"secret/data/foo"},
#  "response":{"data":{"data":{"password":"hmac-sha256:..."}}}}

# Disable HMAC for non-secret keys to make logs queryable
vault audit enable file \
  file_path=/var/log/vault/audit.log \
  hmac_accessor=true
vault read sys/internal/ui/mounts
vault write sys/audit-hash/file \
  input=mysecret      # check what hash a value WOULD produce
```

### Rotation

```bash
# SIGHUP reopens log file (logrotate copytruncate alternative)
kill -HUP $(pidof vault)
```

## Vault Agent

Sidecar that auto-authenticates and serves secrets to the application.

### Components

- Auto-auth: configures one auth method; agent maintains a token in the background.
- Sink: writes the token to a file/HTTP endpoint where consumers can read it.
- Templating: renders Go templates against Vault, writes to disk, reloads consumers.
- Cache: local API cache + token lease tracking.
- Exec: runs a child process with templated env vars.

### Run

```bash
vault agent -config=agent.hcl
vault agent -config=agent.hcl -log-level=debug
```

### Example HCL

```bash
pid_file = "/var/run/vault-agent.pid"
exit_after_auth = false

vault {
  address = "https://vault.example.com:8200"
  retry { num_retries = 5 }
  ca_cert = "/etc/vault.d/ca.pem"
}

auto_auth {
  method "approle" {
    mount_path = "auth/approle"
    config = {
      role_id_file_path   = "/etc/vault-agent/role-id"
      secret_id_file_path = "/etc/vault-agent/secret-id"
      remove_secret_id_file_after_reading = false
    }
  }
  sink "file" {
    config = { path = "/run/vault/token" }
    wrap_ttl = "5m"
  }
}

cache {
  use_auto_auth_token = true
}

listener "tcp" {
  address     = "127.0.0.1:8100"
  tls_disable = true
}

template {
  source      = "/etc/vault-agent/db.tpl"
  destination = "/run/secrets/db.env"
  perms       = "0600"
  command     = "systemctl reload app"
}
```

### Auto-Auth Methods

```bash
# Kubernetes
auto_auth {
  method "kubernetes" {
    mount_path = "auth/kubernetes"
    config = {
      role           = "web-app"
      token_path     = "/var/run/secrets/kubernetes.io/serviceaccount/token"
    }
  }
}

# AWS
auto_auth {
  method "aws" {
    config = { type = "iam" role = "ec2-app" }
  }
}

# JWT
auto_auth {
  method "jwt" {
    config = {
      path = "/var/run/secrets/tokens/vault"
      role = "gha-deploy"
    }
  }
}
```

## Vault Agent Templating

Go templates with Vault-specific functions. Renders to disk; rerenders when leases approach expiry.

```bash
template {
  source      = "/etc/vault-agent/db.tpl"
  destination = "/run/secrets/db.env"
  perms       = "0640"
  contents    = ""        # alternative to source: inline template
  command     = "systemctl reload app"
  command_timeout = "30s"
  error_on_missing_key = true
  exec {
    command = ["/usr/bin/myapp"]
    restart_on_secret_changes = "always"
    restart_kill_signal = "SIGTERM"
  }
  wait {
    min = "5s"
    max = "30s"
  }
  left_delimiter  = "{{"
  right_delimiter = "}}"
}
```

### Functions

```bash
# Read kv-v2
{{ with secret "secret/data/db/postgres" }}
DB_USER={{ .Data.data.username }}
DB_PASS={{ .Data.data.password }}
{{ end }}

# Read kv-v1
{{ with secret "kv1/db/postgres" }}
DB_PASS={{ .Data.password }}
{{ end }}

# Dynamic creds — re-renders before lease expiry
{{ with secret "database/creds/readonly" }}
DB_USER={{ .Data.username }}
DB_PASS={{ .Data.password }}
{{ end }}

# PKI
{{ with pkiCert "pki/issue/web" "common_name=app.example.com" "ttl=72h" }}
{{ .Data.private_key }}
{{ .Data.certificate }}
{{ .Data.issuing_ca }}
{{ end }}

# Loops
{{ range secrets "secret/metadata/team-a/" }}
{{ . }}
{{ end }}
```

### Reload Signals

```bash
template {
  command = "systemctl reload app"     # POSIX shell
}
# Or send a signal directly via exec stanza
```

## Operator Commands

### generate-root

Replaces an unrecoverably lost root token. Requires unseal/recovery key threshold.

```bash
vault operator generate-root -init -otp=$(vault operator generate-root -generate-otp)
# Distributes nonce + OTP

# Each operator with a key:
vault operator generate-root
# Enter nonce, then unseal key

vault operator generate-root -decode=$ENCODED_TOKEN -otp=$OTP

vault operator generate-root -cancel
vault operator generate-root -status

# DR operation token
vault operator generate-root -dr-token
```

### step-down

Force the active node to relinquish HA leadership.

```bash
vault operator step-down
```

### rotate

Rotate the underlying encryption key. Old data still decryptable.

```bash
vault operator rotate
vault read sys/key-status   # Term and InstallTime
```

### rekey

Rotate the unseal/recovery key shares.

```bash
vault operator rekey -init -key-shares=5 -key-threshold=3
# Prints nonce

# Each operator
vault operator rekey -nonce=$NONCE $UNSEAL_KEY

vault operator rekey -status
vault operator rekey -cancel

# Recovery rekey (auto-unseal)
vault operator rekey -target=recovery -init -key-shares=5 -key-threshold=3
```

### Raft

```bash
# Save snapshot
vault operator raft snapshot save snapshot.snap

# Restore (DESTRUCTIVE — replaces existing data)
vault operator raft snapshot restore snapshot.snap
vault operator raft snapshot restore -force snapshot.snap

# List peers
vault operator raft list-peers

# Add new node (manual; usually via retry_join in config)
vault operator raft join https://vault-1:8200

# Remove peer
vault operator raft remove-peer vault-3

# Autopilot status (server health, dead servers, voter promotion)
vault operator raft autopilot state
vault operator raft autopilot get-config
vault operator raft autopilot set-config \
  -cleanup-dead-servers=true \
  -last-contact-threshold=10s \
  -dead-server-last-contact-threshold=24h \
  -min-quorum=3 \
  -server-stabilization-time=10s
```

## HCL Server Config

```bash
storage "raft" {
  path    = "/opt/vault/data"
  node_id = "vault-1"
  performance_multiplier = 1
  retry_join {
    leader_api_addr        = "https://vault-2:8200"
    leader_ca_cert_file    = "/etc/vault.d/ca.pem"
    leader_client_cert_file= "/etc/vault.d/client.pem"
    leader_client_key_file = "/etc/vault.d/client-key.pem"
  }
}

listener "tcp" {
  address                            = "0.0.0.0:8200"
  cluster_address                    = "0.0.0.0:8201"
  tls_cert_file                      = "/etc/vault.d/server.pem"
  tls_key_file                       = "/etc/vault.d/server-key.pem"
  tls_min_version                    = "tls12"
  tls_cipher_suites                  = "TLS_ECDHE_..."
  tls_require_and_verify_client_cert = true
  tls_client_ca_file                 = "/etc/vault.d/client-ca.pem"
  tls_disable_client_certs           = false
  x_forwarded_for_authorized_addrs   = "10.0.0.0/8"
  http_idle_timeout                  = "5m"
  http_read_header_timeout           = "10s"
  http_read_timeout                  = "30s"
  http_write_timeout                 = "0"
  max_request_size                   = 33554432
  max_request_duration               = "90s"
}

listener "unix" {
  address = "/run/vault.sock"
}

api_addr     = "https://vault-1.example.com:8200"
cluster_addr = "https://vault-1.internal:8201"
cluster_name = "vault-prod"

ui = true

default_lease_ttl = "768h"
max_lease_ttl     = "8760h"

disable_mlock = false   # set true on AWS/Azure if Vault crashes due to mlock limits
disable_clustering = false
disable_cache = false
disable_printable_check = false
log_level = "info"
log_format = "json"
log_file = "/var/log/vault/vault.log"
log_rotate_duration = "24h"
log_rotate_max_files = 30

pid_file = "/var/run/vault.pid"

raw_storage_endpoint = false

telemetry {
  prometheus_retention_time = "24h"
  disable_hostname          = true
  statsd_address            = "localhost:8125"
  dogstatsd_addr            = "localhost:8125"
  metrics_prefix            = "vault"
  enable_hostname_label     = false
  usage_gauge_period        = "10m"
  maximum_gauge_cardinality = 500
}

# Enterprise license
license_path = "/etc/vault.d/license.hclic"
```

`disable_mlock = true` is REQUIRED on filesystems that don't support `mlock` (NFS, some containers). The trade-off: encryption keys may swap to disk. Mitigate by disabling swap on the host.

## HTTP API

All operations have an HTTP endpoint. Auth via `X-Vault-Token`.

### Headers

```bash
X-Vault-Token: hvs.CAESI...
X-Vault-Namespace: team-a       # Enterprise
X-Vault-Wrap-TTL: 5m            # Request response wrapping
X-Vault-Request: 1              # Detect proxies stripping body
X-Vault-Index: ...              # Replication consistency
X-Vault-Forward: active-node    # Force forwarding to active
X-Vault-Inconsistent: forward-active-node | fail
```

### Endpoints

```bash
# System
GET  /v1/sys/health              # 200 active, 429 standby, 472 DR sec, 473 perf sec, 501 not init, 503 sealed
GET  /v1/sys/seal-status
PUT  /v1/sys/unseal              # body: { "key": "..." }
PUT  /v1/sys/seal
GET  /v1/sys/init
PUT  /v1/sys/init                # body: { "secret_shares":5, "secret_threshold":3 }

GET  /v1/sys/mounts
POST /v1/sys/mounts/<path>       # body: { "type":"kv", "options":{"version":"2"} }
DELETE /v1/sys/mounts/<path>

GET  /v1/sys/auth
POST /v1/sys/auth/<path>         # body: { "type":"userpass" }

GET  /v1/sys/policies/acl
PUT  /v1/sys/policies/acl/<name> # body: { "policy":"path \"...\" {...}" }

GET  /v1/sys/leases/lookup/<prefix>
PUT  /v1/sys/leases/renew        # body: { "lease_id":"..." }
PUT  /v1/sys/leases/revoke       # body: { "lease_id":"..." }

# Auth
POST /v1/auth/token/create       # body: { "policies":["..."], "ttl":"1h" }
POST /v1/auth/token/lookup       # body: { "token":"..." }
POST /v1/auth/token/revoke       # body: { "token":"..." }
POST /v1/auth/userpass/login/<u> # body: { "password":"..." }
POST /v1/auth/approle/login      # body: { "role_id":"...", "secret_id":"..." }

# kv-v2
GET  /v1/secret/data/foo
POST /v1/secret/data/foo         # body: { "data":{"k":"v"}, "options":{"cas":3} }
GET  /v1/secret/metadata/foo
LIST /v1/secret/metadata/foo
DELETE /v1/secret/data/foo
POST /v1/secret/destroy/foo      # body: { "versions":[1,2] }

# kv-v1
GET  /v1/kv1/foo
POST /v1/kv1/foo                  # body: { "key":"value" }
DELETE /v1/kv1/foo
```

### Curl Examples

```bash
# Login
curl -sS -X POST -d '{"password":"s3cret"}' \
  $VAULT_ADDR/v1/auth/userpass/login/alice | jq

# Read kv-v2
curl -sS -H "X-Vault-Token: $VAULT_TOKEN" \
  $VAULT_ADDR/v1/secret/data/db/postgres | jq -r .data.data.password

# Write kv-v2
curl -sS -X POST -H "X-Vault-Token: $VAULT_TOKEN" \
  -d '{"data":{"username":"app","password":"s3cret"}}' \
  $VAULT_ADDR/v1/secret/data/db/postgres

# List
curl -sS -X LIST -H "X-Vault-Token: $VAULT_TOKEN" \
  $VAULT_ADDR/v1/secret/metadata/db

# Health (no auth)
curl -sS $VAULT_ADDR/v1/sys/health
```

### LIST quirk

LIST is not standard HTTP. Vault accepts `GET ?list=true` as fallback.

```bash
curl -sS -H "X-Vault-Token: $VAULT_TOKEN" \
  "$VAULT_ADDR/v1/secret/metadata/db?list=true"
```

## Response Wrapping

Wrap a response so the recipient unwraps once and only once. The wrapping token has its own TTL and lives in the cubbyhole — even Vault root cannot read the wrapped content.

### Wrap

```bash
# Wrap any response
vault read -wrap-ttl=5m secret/data/db/postgres
# wrapping_token        hvs.CAESI...
# wrapping_accessor     ...
# wrapping_token_ttl    5m

# Wrap a generated SecretID
vault write -wrap-ttl=60s -f auth/approle/role/app/secret-id

# Sys-level wrap of arbitrary JSON
vault write -wrap-ttl=10m sys/wrapping/wrap \
  payload=@cargo.json
```

### Unwrap

```bash
# As the recipient (token = wrapping_token)
VAULT_TOKEN=$WRAP vault unwrap

# Without consuming current token (sys/wrapping/unwrap)
vault write sys/wrapping/unwrap token=$WRAP
```

### Inspect (without consuming)

```bash
# Lookup metadata only
vault write sys/wrapping/lookup token=$WRAP
# creation_path, creation_time, creation_ttl

# Rewrap (extend TTL via re-issue)
vault write sys/wrapping/rewrap token=$WRAP
```

## Common Errors

### permission denied (Code 403)

```bash
# Error making API request. URL: ... Code: 403. Errors: * permission denied
```

Cause: token lacks the capability for that path; or token is in a namespace the path isn't in; or KV-v2 path is missing `data/` in the policy.

Fix: confirm policy includes the right path/capability.

```bash
vault token capabilities secret/data/foo
vault token lookup
# Verify policies attached
vault policy read app-read
```

### server is sealed

```bash
# Error: server is sealed
```

Cause: Vault has just started or was sealed manually.

Fix: unseal.

```bash
vault status
vault operator unseal $KEY1
vault operator unseal $KEY2
vault operator unseal $KEY3
```

### vault is initialized

```bash
# Error: vault is initialized
```

Cause: trying to `vault operator init` an already-initialized cluster.

Fix: do not init twice. Use existing root or `operator generate-root` if root is lost.

### missing client token

```bash
# Error: missing client token
```

Cause: `VAULT_TOKEN` unset, no `~/.vault-token`, or call to `auth/...` requiring no token but malformed.

Fix:

```bash
export VAULT_TOKEN=$(vault login -method=userpass -token-only username=alice)
# or
vault login -method=oidc role=reader
```

### insufficient capabilities

```bash
# Error: token has insufficient capabilities for this operation
```

Cause: token has read but tried to update, or path requires sudo.

Fix: review policy; some endpoints (sys/raw, generate-root) demand sudo.

```bash
path "sys/raw/*" {
  capabilities = ["read","sudo"]
}
```

### 404 reading kv

```bash
# Error reading secret/data/X: Code: 404. Errors:
```

Cause #1: kv-v1 vs kv-v2 path confusion. KV-v2 needs `vault kv get secret/X` (CLI hides `data/`); raw API call to `secret/X` returns 404 because the real path is `secret/data/X`.

Cause #2: secret legitimately deleted/destroyed.

Fix:

```bash
vault secrets list -detailed | grep ^secret/   # check version
# kv_v2: use 'vault kv get secret/X'  (CLI sugar)
# Or hit  GET /v1/secret/data/X  via API
```

### cluster_addr is required

```bash
# Error: cluster_addr is required
```

Cause: `cluster_addr` is empty in HCL when storage type requires HA (raft, consul).

Fix:

```bash
api_addr     = "https://vault-1.example.com:8200"
cluster_addr = "https://vault-1.internal:8201"
```

### invalid OIDC discovery URL

```bash
# Error: invalid OIDC discovery URL
```

Cause: `oidc_discovery_url` is unreachable, returns non-200, or has a TLS error; or `bound_issuer` mismatches what the IdP advertises.

Fix:

```bash
curl -sS https://accounts.example.com/.well-known/openid-configuration | jq .issuer
# Confirm discovery doc is reachable AND issuer string matches your config
vault read auth/oidc/config
```

### error decrypting wrapped response

```bash
# Error: error decrypting wrapped response: invalid input given
```

Cause: wrapping token already used (single-use), expired, or the response_wrapping_token field was double-handled.

Fix:

```bash
vault write sys/wrapping/lookup token=$WRAP   # check if still valid
# Re-wrap if needed (creator side)
vault write -wrap-ttl=5m -f auth/approle/role/app/secret-id
```

### replication routing error

```bash
# Error: this server is part of a replication cluster, please use the standby server
```

Cause: writing to a perf secondary endpoint that doesn't forward; or attempting writes on a DR secondary (which never accepts writes).

Fix: target the primary, or set `VAULT_DISABLE_REDIRECTS=false` so the client follows the redirect.

```bash
vault read sys/replication/status
# Determine primary, point VAULT_ADDR there
```

## Common Gotchas

### KV-v2 path missing data/ in policy

```bash
# BROKEN: 403 permission denied
path "secret/team-a/*" { capabilities = ["read"] }

# FIXED
path "secret/data/team-a/*"     { capabilities = ["read"] }
path "secret/metadata/team-a/*" { capabilities = ["list","read"] }
```

### CLI vs API endpoint mismatch

```bash
# BROKEN: API path lacks data/
curl -H "X-Vault-Token: $T" $VAULT_ADDR/v1/secret/db/postgres
# 404

# FIXED
curl -H "X-Vault-Token: $T" $VAULT_ADDR/v1/secret/data/db/postgres
```

### VAULT_ADDR with TLS verification

```bash
# BROKEN: self-signed cert; client refuses to connect
export VAULT_ADDR=https://vault.example.com:8200

# FIXED: tell client where the CA lives
export VAULT_CACERT=/etc/vault.d/ca.pem
# OR (DANGEROUS, dev only)
export VAULT_SKIP_VERIFY=true
```

### Orphan vs default child

```bash
# BROKEN: parent token revoked; service token revoked too
PARENT=$(vault token create -policy=app -format=json | jq -r .auth.client_token)
SERVICE=$(VAULT_TOKEN=$PARENT vault token create -policy=app \
  -format=json | jq -r .auth.client_token)
# Later: vault token revoke $PARENT  -> SERVICE is also dead

# FIXED: orphan keeps SERVICE alive
SERVICE=$(VAULT_TOKEN=$PARENT vault token create -policy=app -orphan \
  -format=json | jq -r .auth.client_token)
```

### max_lease_ttl truncation

```bash
# BROKEN: token created with -ttl=8760h gets clamped to mount default
vault token create -ttl=8760h -policy=long
# token_duration only 768h

# FIXED: tune mount or use a token role with explicit max
vault auth tune -max-lease-ttl=8760h token/
# Or
vault write auth/token/roles/long allowed_policies=long \
  token_explicit_max_ttl=8760h
vault token create -role=long
```

### LDAP groups not syncing without re-login

```bash
# BROKEN: added user to LDAP group "engineering"; vault still shows old policies
# Vault caches group membership at login time

# FIXED: re-login (or expire token)
vault token revoke -self
vault login -method=ldap username=alice
vault token capabilities secret/data/eng/foo
```

### 768h max for non-period tokens

Tokens default to a system max of `max_lease_ttl = 768h` (32 days). Beyond that, only periodic tokens or tokens via a role with `token_explicit_max_ttl` extend further.

```bash
# BROKEN
vault token create -ttl=2160h
# token_duration 768h (clamped)

# FIXED: periodic
vault token create -period=2160h -policy=app
# Renews to 2160h forever
```

### Wrapping token used twice

```bash
# BROKEN
vault unwrap $WRAP
vault unwrap $WRAP   # Error: error decrypting wrapped response

# FIXED: wrap again (creator only) and hand the new wrapping token
vault write -wrap-ttl=5m -f auth/approle/role/app/secret-id
```

### Vault behind a load balancer without api_addr

```bash
# BROKEN: client gets redirected to internal node IP it cannot reach
# Vault returns Location: https://10.0.0.5:8200 ...
# Client times out

# FIXED: api_addr returns the LB-facing URL
api_addr     = "https://vault.example.com"
cluster_addr = "https://vault-1.internal:8201"
```

### Standby returning 429

```bash
# Looks broken, isn't. /v1/sys/health returns 429 on standby — by design.
# Configure your LB to treat 429 as healthy if you intentionally LB to standbys.
```

### Periodic token still expires

```bash
# BROKEN: agent crashed; token not renewed for 73h on a -period=72h token
# Token revoked — no auto-renew when agent down.

# FIXED: monitor agent + renew sufficiently in advance (Vault Agent default
# renews at ~50% TTL).
```

### VAULT_TOKEN exported across shells

```bash
# BROKEN: forgot to unset; an older root token still usable in another shell
echo $VAULT_TOKEN
# hvs.AAAAAQ... (root!)
# History reveals it; logs reveal it.

# FIXED: scope tokens; use `vault login` per-shell; remove ~/.vault-token
unset VAULT_TOKEN
rm -f ~/.vault-token
```

### disable_mlock=false on systems without mlock

```bash
# BROKEN: Vault crashes on start
# Error: failed to lock memory: cannot allocate memory

# FIXED: either raise system limits OR (with swap disabled)
disable_mlock = true
```

## Idioms

### Health-check in scripts

```bash
# Boolean-style sealed check
if vault status -format=json 2>/dev/null | jq -e '.sealed == true' >/dev/null; then
  echo "Vault is sealed"; exit 1
fi

# /v1/sys/health: HTTP code is the answer
curl -sf -o /dev/null -w '%{http_code}\n' $VAULT_ADDR/v1/sys/health
# 200 active, 429 standby, 472 DR sec, 473 perf sec, 501 not init, 503 sealed
```

### vault read -format=json | jq

```bash
PASS=$(vault kv get -format=json secret/db/postgres | jq -r .data.data.password)
LEASE=$(vault read -format=json database/creds/readonly | jq -r .lease_id)
TOKEN=$(vault write -format=json auth/approle/login \
  role_id=$RID secret_id=$SID | jq -r .auth.client_token)
```

### Vault Agent for non-interactive auth

Don't ship long-lived tokens in env vars. Run `vault agent` as a sidecar and consume the sink file (or template).

```bash
# /etc/systemd/system/myapp.service
[Service]
EnvironmentFile=/run/secrets/db.env
ExecStart=/usr/bin/myapp
```

### Transit as a KMS

Have apps encrypt PII via `transit/encrypt/orders`; store ciphertext anywhere. Decrypt on retrieval. Rotate the key without rewriting old ciphertext (via `rewrap/`).

```bash
PT=$(base64 <<< "card-1234")
CT=$(vault write -field=ciphertext transit/encrypt/orders plaintext=$PT)
# Store $CT in DB. Decrypt later.
```

### PKI as internal CA

Issue 24h-72h certs to services on demand. No more "year-long cert expired in production" pages.

```bash
# Cron / systemd timer / vault agent template
vault write -format=json pki/issue/web common_name=app.example.com ttl=24h \
  | jq -r '.data.certificate, .data.private_key' > /etc/ssl/app.pem
systemctl reload nginx
```

### Database dynamic creds for ephemeral DB users

```bash
read -r USER PASS LEASE < <(vault read -format=json database/creds/readonly \
  | jq -r '[.data.username,.data.password,.lease_id] | @tsv')
psql "host=pg user=$USER password=$PASS dbname=app"
vault lease revoke $LEASE
```

### One-shot SecretID delivery

```bash
WRAP=$(vault write -wrap-ttl=60s -f -format=json \
  auth/approle/role/app/secret-id | jq -r .wrap_info.token)
# Hand $WRAP to consumer over an out-of-band channel.
# Consumer once-only:
SID=$(VAULT_TOKEN=$WRAP vault unwrap -format=json | jq -r .data.secret_id)
```

## Operations

### Raft Snapshots (Backup)

```bash
# Cron daily
vault operator raft snapshot save \
  /var/backups/vault/$(date +%F).snap

# Verify size + non-empty
test -s /var/backups/vault/$(date +%F).snap

# Restore (DESTRUCTIVE)
vault operator raft snapshot restore -force /var/backups/vault/2026-04-25.snap

# Encrypted off-site copy
vault operator raft snapshot save - | age -r age1... > vault.snap.age
```

Snapshots include all storage; encrypted at rest with the master key. Restoring requires a target Vault initialized with the SAME unseal/recovery key material.

### Backup via auto-snapshots (Enterprise)

```bash
vault write sys/storage/raft/snapshot-auto/config/daily \
  interval=24h \
  retain=7 \
  storage_type=aws-s3 \
  aws_s3_bucket=vault-backups \
  aws_s3_region=us-east-1 \
  path_prefix=prod/
```

### Key Rotation Cadence

- `vault operator rotate` — every 6-12 months for the encryption key.
- `vault operator rekey` — annually or after operator turnover; updates unseal/recovery shares.
- `vault write -f transit/keys/<name>/rotate` — every 90 days per key, or after suspected exposure.
- PKI root CA — every 5-10 years; intermediate every 1-3 years.

### Audit Log Rotation

```bash
# /etc/logrotate.d/vault
/var/log/vault/audit.log {
  daily
  rotate 30
  compress
  postrotate
    /bin/kill -HUP $(cat /var/run/vault.pid) 2>/dev/null || true
  endscript
}
```

Vault re-opens the audit file on SIGHUP. Don't use `copytruncate` — it writes-through into a new descriptor and leaves zero-byte log gaps.

### HA Pairing

Three or five voting Raft peers. Two-node clusters cannot tolerate any failure (quorum = 2/2). Use Autopilot to demote unstable nodes.

```bash
vault operator raft list-peers
# Node          Address           State   Voter
# vault-1       10.0.1.5:8201     leader  true
# vault-2       10.0.1.6:8201     follower true
# vault-3       10.0.1.7:8201     follower true

vault operator raft autopilot state
```

### Rolling Upgrades

```bash
# 1. Snapshot
vault operator raft snapshot save pre-upgrade.snap

# 2. Drain a follower (per node):
#    a. systemctl stop vault on follower
#    b. apt-get install vault=NEW_VERSION (or replace binary)
#    c. systemctl start vault
#    d. vault operator unseal (Shamir) — auto-unseal needs no action
#    e. confirm raft list-peers shows it as follower in sync

# 3. Repeat for all followers.

# 4. step-down on the leader, then upgrade it (becomes new follower).
vault operator step-down

# 5. Verify
vault status
vault read sys/health
vault operator raft list-peers
```

Always upgrade by at most 1 minor version at a time. Run the new version against the old data dir for at least a few minutes on each follower before continuing.

### Performance Tuning

```bash
# Raft performance multiplier (1 = best perf, 5 = highest reliability default)
storage "raft" {
  performance_multiplier = 1
}

# HTTP concurrency
listener "tcp" {
  http_idle_timeout        = "5m"
  http_read_header_timeout = "10s"
  http_read_timeout        = "30s"
  http_write_timeout       = "0"
}

# Telemetry to Prometheus
telemetry {
  prometheus_retention_time = "24h"
  disable_hostname          = true
}

# Cache tuning
cache {
  size = "32000"  # entries
}

# Memory
GOMEMLIMIT=4GiB vault server -config=/etc/vault.d/vault.hcl
GOGC=100 vault server -config=...
```

### Health checks

```bash
# Active node only
curl -sf -o /dev/null $VAULT_ADDR/v1/sys/health

# Standby OK
curl -sf -o /dev/null "$VAULT_ADDR/v1/sys/health?standbyok=true"

# Perf standby OK
curl -sf -o /dev/null "$VAULT_ADDR/v1/sys/health?perfstandbyok=true"

# Custom status codes (LB-friendly)
curl -s -o /dev/null -w '%{http_code}' \
  "$VAULT_ADDR/v1/sys/health?activecode=200&standbycode=200&sealedcode=503&uninitcode=501"
```

### Disaster Recovery Drill

```bash
# 1. Take snapshot
vault operator raft snapshot save dr.snap

# 2. Promote DR secondary
vault write sys/replication/dr/secondary/promote \
  dr_operation_token=$OP

# 3. Update DNS / VAULT_ADDR
# 4. Validate apps reconnect
# 5. Re-bootstrap former primary as DR secondary
```

### Lease/Token cleanup

```bash
# Revoke all leases under a prefix (e.g., a leaving team)
vault lease revoke -prefix database/creds/team-a/

# Force revoke (continue past errors)
vault lease revoke -force -prefix /

# List stranded tokens (rare; usually leases own them)
vault list auth/token/accessors
vault token lookup -accessor $ACCESSOR
```

## See Also

- sops
- age
- openssl
- ssh
- gpg
- polyglot

## References

- Official site: https://www.vaultproject.io/
- Tutorials: https://developer.hashicorp.com/vault/tutorials
- Source: https://github.com/hashicorp/vault
- API docs: https://developer.hashicorp.com/vault/api-docs
- CLI docs: https://developer.hashicorp.com/vault/docs/commands
- Server config: https://developer.hashicorp.com/vault/docs/configuration
- Auth methods: https://developer.hashicorp.com/vault/docs/auth
- Secret engines: https://developer.hashicorp.com/vault/docs/secrets
- Policies: https://developer.hashicorp.com/vault/docs/concepts/policies
- Raft storage: https://developer.hashicorp.com/vault/docs/configuration/storage/raft
- Auto-unseal: https://developer.hashicorp.com/vault/docs/concepts/seal
- Vault Agent: https://developer.hashicorp.com/vault/docs/agent-and-proxy/agent
- Replication (Enterprise): https://developer.hashicorp.com/vault/docs/enterprise/replication
- Namespaces (Enterprise): https://developer.hashicorp.com/vault/docs/enterprise/namespaces
- Audit devices: https://developer.hashicorp.com/vault/docs/audit
- Identity: https://developer.hashicorp.com/vault/docs/concepts/identity
