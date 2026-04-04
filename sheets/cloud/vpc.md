# VPC (Virtual Private Cloud)

Isolated virtual network in the cloud with subnets, route tables, gateways, and security controls for hosting workloads with fine-grained network segmentation and connectivity.

## VPC Basics

### Create and describe a VPC (AWS)

```bash
# Create a VPC with a CIDR block
aws ec2 create-vpc --cidr-block 10.0.0.0/16 \
  --tag-specifications 'ResourceType=vpc,Tags=[{Key=Name,Value=prod-vpc}]'

# Describe VPCs
aws ec2 describe-vpcs \
  --query 'Vpcs[].{ID:VpcId,CIDR:CidrBlock,Name:Tags[?Key==`Name`]|[0].Value}' \
  --output table

# Enable DNS hostnames (required for many services)
aws ec2 modify-vpc-attribute \
  --vpc-id vpc-abc123 \
  --enable-dns-hostnames '{"Value": true}'
```

### GCP VPC network

```bash
# Create a custom VPC network
gcloud compute networks create prod-vpc \
  --subnet-mode=custom

# List networks
gcloud compute networks list
```

### Azure VNet

```bash
# Create a virtual network
az network vnet create \
  --name prod-vnet \
  --resource-group my-rg \
  --address-prefixes 10.0.0.0/16

# List VNets
az network vnet list --output table
```

## Subnets

### Public subnet (internet-facing)

```bash
# Create a public subnet
aws ec2 create-subnet \
  --vpc-id vpc-abc123 \
  --cidr-block 10.0.1.0/24 \
  --availability-zone us-east-1a \
  --tag-specifications 'ResourceType=subnet,Tags=[{Key=Name,Value=public-1a}]'

# Enable auto-assign public IPs
aws ec2 modify-subnet-attribute \
  --subnet-id subnet-pub1 \
  --map-public-ip-on-launch
```

### Private subnet (no direct internet)

```bash
# Create a private subnet
aws ec2 create-subnet \
  --vpc-id vpc-abc123 \
  --cidr-block 10.0.10.0/24 \
  --availability-zone us-east-1a \
  --tag-specifications 'ResourceType=subnet,Tags=[{Key=Name,Value=private-1a}]'

# GCP — create a subnet with private Google access
gcloud compute networks subnets create private-subnet \
  --network=prod-vpc \
  --range=10.0.10.0/24 \
  --region=us-east1 \
  --enable-private-ip-google-access
```

## Internet and NAT Gateways

### Internet gateway

```bash
# Create and attach an internet gateway
aws ec2 create-internet-gateway \
  --tag-specifications 'ResourceType=internet-gateway,Tags=[{Key=Name,Value=prod-igw}]'
aws ec2 attach-internet-gateway \
  --internet-gateway-id igw-abc123 \
  --vpc-id vpc-abc123
```

### NAT gateway (outbound-only internet for private subnets)

```bash
# Allocate an Elastic IP for NAT
aws ec2 allocate-address --domain vpc

# Create a NAT gateway in a public subnet
aws ec2 create-nat-gateway \
  --subnet-id subnet-pub1 \
  --allocation-id eipalloc-abc123 \
  --tag-specifications 'ResourceType=natgateway,Tags=[{Key=Name,Value=prod-nat}]'

# Wait for NAT gateway to become available
aws ec2 wait nat-gateway-available --nat-gateway-ids nat-abc123
```

### GCP Cloud NAT

```bash
# Create a Cloud Router (required for Cloud NAT)
gcloud compute routers create prod-router \
  --network=prod-vpc \
  --region=us-east1

# Create Cloud NAT
gcloud compute routers nats create prod-nat \
  --router=prod-router \
  --region=us-east1 \
  --nat-all-subnet-ip-ranges \
  --auto-allocate-nat-external-ips
```

## Route Tables

### Create and associate routes

```bash
# Create a route table
aws ec2 create-route-table --vpc-id vpc-abc123 \
  --tag-specifications 'ResourceType=route-table,Tags=[{Key=Name,Value=public-rt}]'

# Add a default route to IGW (makes it a public route table)
aws ec2 create-route \
  --route-table-id rtb-abc123 \
  --destination-cidr-block 0.0.0.0/0 \
  --gateway-id igw-abc123

# Add a route to NAT for private subnets
aws ec2 create-route \
  --route-table-id rtb-private \
  --destination-cidr-block 0.0.0.0/0 \
  --nat-gateway-id nat-abc123

# Associate route table with a subnet
aws ec2 associate-route-table \
  --route-table-id rtb-abc123 \
  --subnet-id subnet-pub1

# Describe routes
aws ec2 describe-route-tables \
  --route-table-ids rtb-abc123 \
  --query 'RouteTables[].Routes[]' --output table
```

## Security Groups and NACLs

### Security groups (stateful)

```bash
# Create a security group
aws ec2 create-security-group \
  --group-name web-sg \
  --description "Web tier security group" \
  --vpc-id vpc-abc123

# Allow inbound HTTP/HTTPS
aws ec2 authorize-security-group-ingress \
  --group-id sg-abc123 \
  --ip-permissions \
    '[{"IpProtocol":"tcp","FromPort":80,"ToPort":80,"IpRanges":[{"CidrIp":"0.0.0.0/0"}]},
      {"IpProtocol":"tcp","FromPort":443,"ToPort":443,"IpRanges":[{"CidrIp":"0.0.0.0/0"}]}]'

# Allow inbound from another security group
aws ec2 authorize-security-group-ingress \
  --group-id sg-backend \
  --protocol tcp --port 5432 \
  --source-group sg-abc123
```

### Network ACLs (stateless)

```bash
# Create a NACL
aws ec2 create-network-acl --vpc-id vpc-abc123

# Add inbound rule (allow HTTP)
aws ec2 create-network-acl-entry \
  --network-acl-id acl-abc123 \
  --rule-number 100 \
  --protocol tcp --port-range From=80,To=80 \
  --cidr-block 0.0.0.0/0 \
  --rule-action allow --ingress

# Add outbound rule (allow ephemeral ports)
aws ec2 create-network-acl-entry \
  --network-acl-id acl-abc123 \
  --rule-number 100 \
  --protocol tcp --port-range From=1024,To=65535 \
  --cidr-block 0.0.0.0/0 \
  --rule-action allow --egress
```

## VPC Peering

### Peer two VPCs

```bash
# Create a peering connection
aws ec2 create-vpc-peering-connection \
  --vpc-id vpc-abc123 \
  --peer-vpc-id vpc-def456 \
  --peer-region us-west-2

# Accept the peering connection (from the peer account/region)
aws ec2 accept-vpc-peering-connection \
  --vpc-peering-connection-id pcx-abc123

# Add routes for peering traffic
aws ec2 create-route \
  --route-table-id rtb-abc123 \
  --destination-cidr-block 10.1.0.0/16 \
  --vpc-peering-connection-id pcx-abc123
```

## VPN and Direct Connect

### Site-to-site VPN

```bash
# Create a virtual private gateway
aws ec2 create-vpn-gateway --type ipsec.1
aws ec2 attach-vpn-gateway \
  --vpn-gateway-id vgw-abc123 --vpc-id vpc-abc123

# Create a customer gateway (your on-prem router)
aws ec2 create-customer-gateway \
  --type ipsec.1 --public-ip 203.0.113.1 --bgp-asn 65000

# Create the VPN connection
aws ec2 create-vpn-connection \
  --type ipsec.1 \
  --vpn-gateway-id vgw-abc123 \
  --customer-gateway-id cgw-abc123 \
  --options '{"StaticRoutesOnly":false}'
```

## VPC Flow Logs

### Enable and query flow logs

```bash
# Enable flow logs to CloudWatch
aws ec2 create-flow-logs \
  --resource-type VPC \
  --resource-ids vpc-abc123 \
  --traffic-type ALL \
  --log-destination-type cloud-watch-logs \
  --log-group-name vpc-flow-logs \
  --deliver-logs-permission-arn arn:aws:iam::123456789012:role/flow-logs-role

# Enable flow logs to S3
aws ec2 create-flow-logs \
  --resource-type VPC \
  --resource-ids vpc-abc123 \
  --traffic-type ALL \
  --log-destination-type s3 \
  --log-destination arn:aws:s3:::my-flow-logs-bucket

# Query flow logs with CloudWatch Insights
aws logs start-query \
  --log-group-name vpc-flow-logs \
  --start-time $(date -d '1 hour ago' +%s) \
  --end-time $(date +%s) \
  --query-string 'filter action="REJECT" | stats count() by srcAddr, dstPort'
```

## VPC Endpoints

### Gateway and interface endpoints

```bash
# Gateway endpoint for S3 (free, route-table based)
aws ec2 create-vpc-endpoint \
  --vpc-id vpc-abc123 \
  --service-name com.amazonaws.us-east-1.s3 \
  --route-table-ids rtb-private

# Interface endpoint for STS (ENI-based, private DNS)
aws ec2 create-vpc-endpoint \
  --vpc-id vpc-abc123 \
  --vpc-endpoint-type Interface \
  --service-name com.amazonaws.us-east-1.sts \
  --subnet-ids subnet-priv1 \
  --security-group-ids sg-endpoint

# List endpoints
aws ec2 describe-vpc-endpoints --output table
```

## Tips

- Always use multiple AZs for subnets (min 2) for high availability.
- NAT gateways are per-AZ; deploy one in each AZ for resilience.
- Security groups are stateful (return traffic auto-allowed); NACLs are stateless (explicit both ways).
- Use VPC endpoints for S3 and DynamoDB to avoid NAT gateway data processing charges.
- Enable VPC flow logs on all VPCs for security auditing and troubleshooting.
- Non-overlapping CIDR blocks are required for VPC peering — plan address space early.
- Use /16 for VPCs and /24 for subnets as a starting point; avoid /28 (too small for most workloads).
- Tag subnets with "Tier=public" or "Tier=private" for automation and IaC.
- VPC peering is non-transitive — use Transit Gateway for hub-and-spoke topologies.
- Use PrivateLink (interface endpoints) to expose services without crossing the public internet.
- Default security group allows all outbound; restrict it immediately in production.
- Check route table associations when debugging connectivity — subnets inherit the main route table if not explicitly associated.

## See Also

cloud-dns, iam, aws-cli, terraform, gcloud

## References

- [AWS VPC User Guide](https://docs.aws.amazon.com/vpc/latest/userguide/)
- [AWS VPC Subnets](https://docs.aws.amazon.com/vpc/latest/userguide/configure-subnets.html)
- [AWS NAT Gateway](https://docs.aws.amazon.com/vpc/latest/userguide/vpc-nat-gateway.html)
- [AWS VPC Peering](https://docs.aws.amazon.com/vpc/latest/peering/)
- [AWS VPC Flow Logs](https://docs.aws.amazon.com/vpc/latest/userguide/flow-logs.html)
- [GCP VPC Documentation](https://cloud.google.com/vpc/docs/overview)
- [Azure VNet Documentation](https://learn.microsoft.com/en-us/azure/virtual-network/virtual-networks-overview)
- [RFC 1918 — Private Address Space](https://datatracker.ietf.org/doc/html/rfc1918)
