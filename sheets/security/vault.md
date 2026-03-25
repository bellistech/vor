# Vault (HashiCorp Secrets Management)

> Centralized secrets management with dynamic credentials, encryption as a service, and identity-based access.

## Basics

### Initialize and Unseal

```bash
# Initialize Vault (first time only)
vault operator init -key-shares=5 -key-threshold=3

# Unseal (requires threshold number of keys)
vault operator unseal <key-1>
vault operator unseal <key-2>
vault operator unseal <key-3>

# Check seal status
vault status

# Seal Vault (emergency)
vault operator seal
```

### Login

```bash
# Token auth
vault login <token>
vault login -method=token token=<token>

# Userpass auth
vault login -method=userpass username=admin

# AppRole auth
vault write auth/approle/login role_id=<role-id> secret_id=<secret-id>
```

## Secrets Engines

### KV (Key-Value) v2

```bash
# Enable KV v2
vault secrets enable -path=secret kv-v2

# Write a secret
vault kv put secret/myapp db_password=hunter2 api_key=abc123

# Read a secret
vault kv get secret/myapp
vault kv get -field=db_password secret/myapp   # Single field
vault kv get -format=json secret/myapp          # JSON output

# List secrets
vault kv list secret/

# Delete (soft delete — version is marked deleted)
vault kv delete secret/myapp

# Undelete a version
vault kv undelete -versions=2 secret/myapp

# Permanently destroy a version
vault kv destroy -versions=1,2 secret/myapp

# View version metadata
vault kv metadata get secret/myapp
```

### Transit (Encryption as a Service)

```bash
# Enable transit
vault secrets enable transit

# Create encryption key
vault write -f transit/keys/my-key

# Encrypt data (input must be base64-encoded)
vault write transit/encrypt/my-key plaintext=$(echo -n "secret" | base64)

# Decrypt data
vault write transit/decrypt/my-key ciphertext=vault:v1:...
# Decode the returned base64 plaintext
echo "<base64-plaintext>" | base64 -d

# Rotate encryption key
vault write -f transit/keys/my-key/rotate

# Rewrap ciphertext with latest key version
vault write transit/rewrap/my-key ciphertext=vault:v1:...
```

### PKI (Certificate Authority)

```bash
# Enable PKI engine
vault secrets enable pki
vault secrets tune -max-lease-ttl=87600h pki

# Generate root CA
vault write pki/root/generate/internal \
  common_name="My Root CA" ttl=87600h

# Create a role
vault write pki/roles/web-server \
  allowed_domains="example.com" \
  allow_subdomains=true \
  max_ttl=720h

# Issue a certificate
vault write pki/issue/web-server \
  common_name="app.example.com" ttl=72h
```

### Database

```bash
# Enable database engine
vault secrets enable database

# Configure PostgreSQL connection
vault write database/config/mydb \
  plugin_name=postgresql-database-plugin \
  connection_url="postgresql://{{username}}:{{password}}@db:5432/mydb" \
  allowed_roles="readonly" \
  username="vault_admin" \
  password="admin_pass"

# Create a role (dynamic credentials)
vault write database/roles/readonly \
  db_name=mydb \
  creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; GRANT SELECT ON ALL TABLES IN SCHEMA public TO \"{{name}}\";" \
  default_ttl=1h max_ttl=24h

# Get dynamic credentials
vault read database/creds/readonly
```

## Auth Methods

### Enable and Configure

```bash
# Enable auth methods
vault auth enable userpass
vault auth enable ldap
vault auth enable approle

# Userpass — create user
vault write auth/userpass/users/admin password=changeme policies=admin

# AppRole — create role
vault write auth/approle/role/my-app \
  secret_id_ttl=10m token_ttl=20m token_max_ttl=30m \
  policies="my-app-policy"

# Get role-id and secret-id
vault read auth/approle/role/my-app/role-id
vault write -f auth/approle/role/my-app/secret-id

# LDAP configuration
vault write auth/ldap/config \
  url="ldap://ldap.example.com" \
  userdn="ou=Users,dc=example,dc=com" \
  groupdn="ou=Groups,dc=example,dc=com" \
  groupattr="cn"
```

## Policies

### Policy Syntax (HCL)

```hcl
# Read-only access to secrets
path "secret/data/myapp/*" {
  capabilities = ["read", "list"]
}

# Full access to a specific path
path "secret/data/admin/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

# Deny access explicitly
path "secret/data/root-only/*" {
  capabilities = ["deny"]
}

# Allow token self-management
path "auth/token/renew-self" {
  capabilities = ["update"]
}
```

### Manage Policies

```bash
# Write policy from file
vault policy write my-app-policy policy.hcl

# List policies
vault policy list

# Read policy
vault policy read my-app-policy

# Delete policy
vault policy delete my-app-policy
```

## Agent

### Agent Configuration

```hcl
# vault-agent.hcl
auto_auth {
  method "approle" {
    config = {
      role_id_file_path   = "/etc/vault/role-id"
      secret_id_file_path = "/etc/vault/secret-id"
    }
  }
  sink "file" {
    config = { path = "/tmp/vault-token" }
  }
}

template {
  source      = "/etc/vault/templates/config.ctmpl"
  destination = "/app/config.json"
  command     = "systemctl reload myapp"
}

vault {
  address = "https://vault.example.com:8200"
}
```

```bash
# Run agent
vault agent -config=vault-agent.hcl
```

## Namespaces (Enterprise)

```bash
# Create namespace
vault namespace create team-a

# Target a namespace
VAULT_NAMESPACE=team-a vault kv get secret/app

# List namespaces
vault namespace list
```

## Tips

- Never store unseal keys together; distribute among different operators.
- Use AppRole for machine authentication; userpass/LDAP for humans.
- Enable audit logging: `vault audit enable file file_path=/var/log/vault_audit.log`.
- Transit engine avoids the need for application-level crypto libraries.
- Set short TTLs on dynamic database credentials to limit blast radius.
- Use response wrapping for secure secret delivery: `vault kv get -wrap-ttl=5m secret/myapp`.

## References

- [Vault Documentation](https://developer.hashicorp.com/vault/docs)
- [Vault API Reference](https://developer.hashicorp.com/vault/api-docs)
- [Vault Tutorials](https://developer.hashicorp.com/vault/tutorials)
- [Vault Secrets Engines](https://developer.hashicorp.com/vault/docs/secrets)
- [Vault Auth Methods](https://developer.hashicorp.com/vault/docs/auth)
- [Vault PKI Secrets Engine](https://developer.hashicorp.com/vault/docs/secrets/pki)
- [Vault Transit Secrets Engine](https://developer.hashicorp.com/vault/docs/secrets/transit)
- [Vault Architecture](https://developer.hashicorp.com/vault/docs/internals/architecture)
- [Vault Security Model](https://developer.hashicorp.com/vault/docs/internals/security)
- [Vault Production Hardening](https://developer.hashicorp.com/vault/tutorials/operations/production-hardening)
