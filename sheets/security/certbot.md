# Certbot (Let's Encrypt Client)

Obtain, renew, and manage free TLS certificates from Let's Encrypt.

## Obtain Certificates

### Standalone (Certbot Runs Its Own Web Server)

```bash
# Certbot temporarily binds to port 80 -- stop your web server first
sudo certbot certonly --standalone -d acme.com -d www.acme.com
```

### Webroot (Use Existing Web Server)

```bash
# Web server must serve /.well-known/acme-challenge/ from the webroot
sudo certbot certonly --webroot -w /var/www/html -d acme.com -d www.acme.com
```

### Nginx Plugin (Auto-Configures Nginx)

```bash
sudo certbot --nginx -d acme.com -d www.acme.com

# Certificate only, don't modify nginx config
sudo certbot certonly --nginx -d acme.com
```

### Apache Plugin

```bash
sudo certbot --apache -d acme.com -d www.acme.com
```

### Wildcard Certificate (Requires DNS Challenge)

```bash
sudo certbot certonly --manual --preferred-challenges dns \
  -d acme.com -d "*.acme.com"

# With Cloudflare DNS plugin (automated)
sudo certbot certonly --dns-cloudflare \
  --dns-cloudflare-credentials /etc/letsencrypt/cloudflare.ini \
  -d acme.com -d "*.acme.com"
```

### Cloudflare Credentials File

```bash
# /etc/letsencrypt/cloudflare.ini
dns_cloudflare_api_token = your-api-token-here
```

```bash
chmod 600 /etc/letsencrypt/cloudflare.ini
```

### Non-Interactive (for Scripts and CI)

```bash
sudo certbot certonly --standalone \
  -d acme.com \
  --non-interactive \
  --agree-tos \
  --email admin@acme.com
```

## Certificate Management

### List Certificates

```bash
sudo certbot certificates
```

### Certificate File Locations

```bash
# After issuance, certs are at:
/etc/letsencrypt/live/acme.com/fullchain.pem   # cert + intermediate
/etc/letsencrypt/live/acme.com/privkey.pem     # private key
/etc/letsencrypt/live/acme.com/cert.pem        # cert only
/etc/letsencrypt/live/acme.com/chain.pem       # intermediate only
```

## Renewal

### Test Renewal (Dry Run)

```bash
sudo certbot renew --dry-run
```

### Renew All Certificates

```bash
sudo certbot renew
```

### Force Renewal of a Specific Certificate

```bash
sudo certbot renew --cert-name acme.com --force-renewal
```

### Auto-Renewal (Systemd Timer)

```bash
# Usually set up automatically; verify:
sudo systemctl list-timers | grep certbot
sudo systemctl status certbot.timer

# Or check the cron job
cat /etc/cron.d/certbot
```

### Renewal Hooks

```bash
# Reload nginx after renewal
sudo certbot renew --deploy-hook "systemctl reload nginx"

# Hooks can also be placed in directories:
# /etc/letsencrypt/renewal-hooks/pre/     - before renewal
# /etc/letsencrypt/renewal-hooks/deploy/  - after successful renewal
# /etc/letsencrypt/renewal-hooks/post/    - after renewal attempt (success or fail)
```

```bash
# /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh
#!/bin/bash
systemctl reload nginx
```

```bash
chmod +x /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh
```

## Revoke and Delete

### Revoke a Certificate

```bash
sudo certbot revoke --cert-name acme.com

# Revoke by cert file path
sudo certbot revoke --cert-path /etc/letsencrypt/live/acme.com/cert.pem

# Revoke with reason
sudo certbot revoke --cert-name acme.com --reason keycompromise
```

### Delete Certificate Files

```bash
sudo certbot delete --cert-name acme.com
```

## Expanding and Modifying

### Add Domains to Existing Certificate

```bash
sudo certbot certonly --nginx \
  -d acme.com -d www.acme.com -d api.acme.com \
  --expand
```

### Change Domain List (Reissue)

```bash
sudo certbot certonly --nginx \
  -d acme.com -d api.acme.com \
  --cert-name acme.com                  # reuses existing cert name
```

## Staging (Testing Without Rate Limits)

```bash
# Use Let's Encrypt staging server (issues untrusted certs)
sudo certbot certonly --standalone \
  -d test.acme.com \
  --staging

# Clean up staging certs before getting real ones
sudo certbot delete --cert-name test.acme.com
```

## Nginx Configuration Example

```bash
server {
    listen 443 ssl http2;
    server_name acme.com www.acme.com;

    ssl_certificate     /etc/letsencrypt/live/acme.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/acme.com/privkey.pem;

    # Recommended SSL settings
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
}

server {
    listen 80;
    server_name acme.com www.acme.com;
    return 301 https://$host$request_uri;
}
```

## Tips

- Let's Encrypt rate limits: 50 certs per domain per week, 5 duplicate certs per week; use `--staging` for testing
- Certificates are valid for 90 days; certbot auto-renews at 60 days by default
- Always use `fullchain.pem` (not `cert.pem`) in server configs; it includes the intermediate CA
- Wildcard certs require DNS-01 challenge -- you need a DNS plugin or manual DNS TXT record
- `--deploy-hook` only runs on successful renewal; `--post-hook` runs regardless
- Files under `/etc/letsencrypt/live/` are symlinks to `/etc/letsencrypt/archive/`; don't copy them
- If port 80 is blocked, use `--preferred-challenges dns` or `--preferred-challenges tls-alpn-01` (port 443)
- Run `certbot renew --dry-run` after any config change to verify renewal will work
- The `certbot` snap package is the recommended install method on Ubuntu 20.04+

## See Also

- tls, pki, openssl, nginx, cryptography

## References

- [Certbot Documentation](https://eff-certbot.readthedocs.io/)
- [Certbot User Guide](https://eff-certbot.readthedocs.io/en/latest/using.html)
- [Certbot Command-Line Options](https://eff-certbot.readthedocs.io/en/latest/cli.html)
- [Let's Encrypt — How It Works](https://letsencrypt.org/how-it-works/)
- [RFC 8555 — ACME Protocol](https://www.rfc-editor.org/rfc/rfc8555)
- [Arch Wiki — Certbot](https://wiki.archlinux.org/title/Certbot)
- [Certbot DNS Plugin Documentation](https://eff-certbot.readthedocs.io/en/latest/using.html#dns-plugins)
- [Let's Encrypt Rate Limits](https://letsencrypt.org/docs/rate-limits/)
- [Red Hat — Managing TLS Certificates with Certbot](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/securing_networks/requesting-certificates-using-rhel-system-roles_configuring-certificates-issued-and-managed-by-certmonger)
