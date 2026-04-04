# Cloud DNS (Managed DNS Services)

Managed DNS services providing authoritative name resolution with hosted zones, routing policies (weighted, latency, geolocation, failover), health checks, and DNSSEC across AWS Route 53, GCP Cloud DNS, and Azure DNS.

## Hosted Zones

### AWS Route 53

```bash
# Create a public hosted zone
aws route53 create-hosted-zone \
  --name example.com \
  --caller-reference "$(date +%s)"

# Create a private hosted zone (VPC-associated)
aws route53 create-hosted-zone \
  --name internal.example.com \
  --caller-reference "$(date +%s)" \
  --vpc VPCRegion=us-east-1,VPCId=vpc-abc123 \
  --hosted-zone-config PrivateZone=true

# List hosted zones
aws route53 list-hosted-zones --output table

# Get zone details (includes NS records for delegation)
aws route53 get-hosted-zone --id Z0123456789
```

### GCP Cloud DNS

```bash
# Create a public managed zone
gcloud dns managed-zones create example-zone \
  --dns-name="example.com." \
  --description="Production DNS zone" \
  --visibility=public

# Create a private zone
gcloud dns managed-zones create internal-zone \
  --dns-name="internal.example.com." \
  --description="Internal DNS" \
  --visibility=private \
  --networks=prod-vpc

# List zones
gcloud dns managed-zones list
```

### Azure DNS

```bash
# Create a public DNS zone
az network dns zone create \
  --name example.com \
  --resource-group my-rg

# Create a private DNS zone
az network private-dns zone create \
  --name internal.example.com \
  --resource-group my-rg

# Link private zone to VNet
az network private-dns link vnet create \
  --name prod-link \
  --zone-name internal.example.com \
  --resource-group my-rg \
  --virtual-network prod-vnet \
  --registration-enabled true
```

## Record Management

### Create and update records (AWS)

```bash
# Create/update an A record via change batch
aws route53 change-resource-record-sets \
  --hosted-zone-id Z0123456789 \
  --change-batch '{
    "Changes": [{
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "app.example.com",
        "Type": "A",
        "TTL": 300,
        "ResourceRecords": [{"Value": "203.0.113.10"}]
      }
    }]
  }'

# Create a CNAME record
aws route53 change-resource-record-sets \
  --hosted-zone-id Z0123456789 \
  --change-batch '{
    "Changes": [{
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "www.example.com",
        "Type": "CNAME",
        "TTL": 300,
        "ResourceRecords": [{"Value": "app.example.com"}]
      }
    }]
  }'

# Alias record (AWS-specific, no TTL — inherits from target)
aws route53 change-resource-record-sets \
  --hosted-zone-id Z0123456789 \
  --change-batch '{
    "Changes": [{
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "example.com",
        "Type": "A",
        "AliasTarget": {
          "HostedZoneId": "Z2FDTNDATAQYW2",
          "DNSName": "d123456.cloudfront.net",
          "EvaluateTargetHealth": true
        }
      }
    }]
  }'

# List records in a zone
aws route53 list-resource-record-sets --hosted-zone-id Z0123456789
```

### GCP record management

```bash
# Start a transaction (batch changes)
gcloud dns record-sets transaction start --zone=example-zone

# Add an A record
gcloud dns record-sets transaction add "203.0.113.10" \
  --name="app.example.com." \
  --ttl=300 --type=A --zone=example-zone

# Add a CNAME record
gcloud dns record-sets transaction add "app.example.com." \
  --name="www.example.com." \
  --ttl=300 --type=CNAME --zone=example-zone

# Execute the transaction
gcloud dns record-sets transaction execute --zone=example-zone

# List records
gcloud dns record-sets list --zone=example-zone
```

### Azure record management

```bash
# Create an A record
az network dns record-set a add-record \
  --zone-name example.com \
  --resource-group my-rg \
  --record-set-name app \
  --ipv4-address 203.0.113.10

# Create a CNAME record
az network dns record-set cname set-record \
  --zone-name example.com \
  --resource-group my-rg \
  --record-set-name www \
  --cname app.example.com
```

## Routing Policies

### Weighted routing (AWS)

```bash
# Canary deployment — 90/10 traffic split
aws route53 change-resource-record-sets \
  --hosted-zone-id Z0123456789 \
  --change-batch '{
    "Changes": [
      {
        "Action": "UPSERT",
        "ResourceRecordSet": {
          "Name": "api.example.com",
          "Type": "A",
          "SetIdentifier": "primary",
          "Weight": 90,
          "TTL": 60,
          "ResourceRecords": [{"Value": "203.0.113.10"}]
        }
      },
      {
        "Action": "UPSERT",
        "ResourceRecordSet": {
          "Name": "api.example.com",
          "Type": "A",
          "SetIdentifier": "canary",
          "Weight": 10,
          "TTL": 60,
          "ResourceRecords": [{"Value": "203.0.113.20"}]
        }
      }
    ]
  }'
```

### Latency-based routing (AWS)

```bash
# Route to the lowest-latency region
aws route53 change-resource-record-sets \
  --hosted-zone-id Z0123456789 \
  --change-batch '{
    "Changes": [
      {
        "Action": "UPSERT",
        "ResourceRecordSet": {
          "Name": "api.example.com", "Type": "A",
          "SetIdentifier": "us-east", "Region": "us-east-1",
          "TTL": 60, "ResourceRecords": [{"Value": "10.0.1.1"}]
        }
      },
      {
        "Action": "UPSERT",
        "ResourceRecordSet": {
          "Name": "api.example.com", "Type": "A",
          "SetIdentifier": "eu-west", "Region": "eu-west-1",
          "TTL": 60, "ResourceRecords": [{"Value": "10.1.1.1"}]
        }
      }
    ]
  }'
```

### Geolocation routing (AWS)

```bash
# Route based on client geography
aws route53 change-resource-record-sets \
  --hosted-zone-id Z0123456789 \
  --change-batch '{
    "Changes": [
      {
        "Action": "UPSERT",
        "ResourceRecordSet": {
          "Name": "app.example.com", "Type": "A",
          "SetIdentifier": "europe",
          "GeoLocation": {"ContinentCode": "EU"},
          "TTL": 300, "ResourceRecords": [{"Value": "10.1.0.1"}]
        }
      },
      {
        "Action": "UPSERT",
        "ResourceRecordSet": {
          "Name": "app.example.com", "Type": "A",
          "SetIdentifier": "default",
          "GeoLocation": {"CountryCode": "*"},
          "TTL": 300, "ResourceRecords": [{"Value": "10.0.0.1"}]
        }
      }
    ]
  }'
```

## Health Checks

### Create and configure health checks

```bash
# HTTP health check
aws route53 create-health-check --caller-reference "$(date +%s)" \
  --health-check-config '{
    "IPAddress": "203.0.113.10",
    "Port": 443,
    "Type": "HTTPS",
    "ResourcePath": "/health",
    "FullyQualifiedDomainName": "api.example.com",
    "RequestInterval": 10,
    "FailureThreshold": 3
  }'

# Calculated health check (combine multiple)
aws route53 create-health-check --caller-reference "$(date +%s)" \
  --health-check-config '{
    "Type": "CALCULATED",
    "ChildHealthChecks": ["hc-id-1", "hc-id-2", "hc-id-3"],
    "HealthThreshold": 2
  }'

# List health checks
aws route53 list-health-checks

# Get health check status
aws route53 get-health-check-status --health-check-id hc-abc123
```

### Failover routing with health checks

```bash
# Primary record with health check
aws route53 change-resource-record-sets \
  --hosted-zone-id Z0123456789 \
  --change-batch '{
    "Changes": [{
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "api.example.com", "Type": "A",
        "SetIdentifier": "primary",
        "Failover": "PRIMARY",
        "HealthCheckId": "hc-abc123",
        "TTL": 60, "ResourceRecords": [{"Value": "203.0.113.10"}]
      }
    },
    {
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "api.example.com", "Type": "A",
        "SetIdentifier": "secondary",
        "Failover": "SECONDARY",
        "TTL": 60, "ResourceRecords": [{"Value": "203.0.113.20"}]
      }
    }]
  }'
```

## DNSSEC

### Enable DNSSEC signing (AWS)

```bash
# Create a KMS key for DNSSEC
aws kms create-key \
  --customer-master-key-spec ECC_NIST_P256 \
  --key-usage SIGN_VERIFY \
  --region us-east-1

# Enable DNSSEC signing
aws route53 create-key-signing-key \
  --hosted-zone-id Z0123456789 \
  --key-management-service-arn arn:aws:kms:us-east-1:123456789012:key/key-id \
  --name my-ksk \
  --status ACTIVE

aws route53 enable-hosted-zone-dnssec --hosted-zone-id Z0123456789

# GCP — enable DNSSEC
gcloud dns managed-zones update example-zone --dnssec-state=on
```

## Tips

- Always set a default/fallback record for geolocation routing to catch unmatched regions.
- Use alias records (AWS) or ANAME records instead of CNAMEs at the zone apex.
- Set TTLs low (60s) during migrations, then increase (300-3600s) for steady state.
- Health checks cost money per check — consolidate with calculated health checks where possible.
- Use private hosted zones with VPC associations for internal service discovery.
- Enable query logging for debugging resolution issues and security auditing.
- DNSSEC adds latency and complexity; enable it only when your threat model requires it.
- Weighted routing with weight=0 removes an endpoint from rotation without deleting the record.
- Use Route 53 Resolver endpoints for hybrid DNS (on-prem to cloud forwarding).
- GCP Cloud DNS transactions ensure atomic multi-record updates — always use them.
- Monitor health check status with CloudWatch alarms to detect failover events.
- Test DNS changes with `dig` or `nslookup` against the authoritative nameservers before relying on propagation.

## See Also

vpc, iam, aws-cli, gcloud, dns

## References

- [AWS Route 53 Developer Guide](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/)
- [AWS Route 53 Routing Policies](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/routing-policy.html)
- [AWS Route 53 Health Checks](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/dns-failover.html)
- [GCP Cloud DNS Documentation](https://cloud.google.com/dns/docs/overview)
- [Azure DNS Documentation](https://learn.microsoft.com/en-us/azure/dns/)
- [RFC 1035 — Domain Names Implementation](https://datatracker.ietf.org/doc/html/rfc1035)
- [RFC 4034 — DNSSEC Resource Records](https://datatracker.ietf.org/doc/html/rfc4034)
