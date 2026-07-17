# AWS Network Architecture & Data Transfer (SAA-C03 Tasks 3.4 + 4.4)

> Design high-performing, scalable, cost-optimized networks: VPC design and routing, endpoints/PrivateLink, peering vs Transit Gateway, Direct Connect vs VPN, CloudFront vs Global Accelerator, hybrid DNS, and the data-transfer cost model.

## VPC Design Fundamentals

```
# CIDR: /16–/28 per VPC; add up to 5 secondary CIDRs (grow without
#   re-IP); plan NON-OVERLAPPING ranges across VPCs/on-prem or
#   peering/VPN routing breaks — IPAM manages allocation at org scale
# Per subnet: AWS reserves 5 IPs (.0 net, .1 router, .2 DNS, .3
#   future, .255 bcast) — /24 yields 251
# Subnet = one AZ; tier design → cs aws-network-security
# Route tables: subnet association (or VPC main); most-specific
#   route wins; local route (VPC CIDR) always present, never removable
# IGW — internet for public subnets (1 per VPC); instance still needs
#   public IP/EIP
# Egress-only IGW — outbound-ONLY for IPv6 (the IPv6 "NAT gateway
#   equivalent" — NAT GW is IPv4-only)
# NAT gateway — managed, per-AZ, 5–100 Gbps auto-scaling, hourly +
#   per-GB processing; SG cannot attach to it
# IPv6: /56 VPC allocation (AWS-provided GUA or BYOIP), all public by
#   design (routing decides exposure); dual-stack standard; IPv6-only
#   subnets exist
# DNS: Route 53 Resolver at VPC+2 (.2); enable DnsSupport +
#   DnsHostnames for private zone use
# VPC Flow Logs — accept/reject metadata (not payload) at VPC/subnet/
#   ENI level → CloudWatch Logs/S3; "who is talking to X / diagnose
#   SG blocks" answer; Traffic Mirroring for actual packets
# Reachability Analyzer — static path analysis ("why can't A reach B")
```

## VPC Endpoints & PrivateLink

```
# Gateway endpoints — S3 + DynamoDB ONLY; route-table prefix-list
#   entry; FREE; same-Region, same-VPC access only
# Interface endpoints (PrivateLink) — ENI with private IP for 100s of
#   AWS services + SaaS + your own services; hourly + per-GB charges;
#   private DNS overrides public names; SGs apply; accessible over
#   DX/VPN/peering (gateway endpoints are NOT)
# Endpoint policies restrict actions/resources through the endpoint
# PrivateLink endpoint SERVICES — expose YOUR app to other VPCs:
#   NLB (or GWLB) behind an endpoint service; consumers create
#   interface endpoints; no route/CIDR coupling, no overlap concerns,
#   one-way exposure only — "SaaS provider offers service privately
#   to 100s of customer VPCs" → PrivateLink (peering/TGW would leak
#   whole networks and hit overlap)
# Cost angle: S3 gateway endpoint is free and kills NAT-processing
#   charges for S3 traffic from private subnets — recurring exam answer
```

## VPC-to-VPC: Peering vs Transit Gateway

```
# VPC Peering — 1:1, non-transitive (A–B, B–C ≠ A–C), no overlapping
#   CIDRs, cross-account + cross-Region OK, no bandwidth choke point,
#   NO transit to VPN/DX/IGW through a peer; route table entries both
#   sides; free within AZ (cross-AZ/Region transfer rates apply)
#   → few VPCs (mesh grows n(n-1)/2 — 10 VPCs = 45 peerings)
# Transit Gateway (TGW) — Regional hub-and-spoke router: 1000s of VPC
#   attachments + VPN + DX gateway + TGW peering (cross-Region,
#   static); route tables per attachment = segmentation (prod vs dev
#   domains); multicast support; 50 Gbps/attachment scale; hourly per
#   attachment + per-GB processing
#   → "simplify connectivity for many VPCs and on-prem" → TGW
# Shared VPC (RAM) — share SUBNETS with other accounts: one network,
#   many accounts — alternative to connecting VPCs at all
# Central egress/inspection: spoke VPCs → TGW → inspection VPC
#   (Network Firewall/GWLB) → egress VPC NAT — the enterprise pattern
# CloudWAN — managed global network layered above TGWs (aware-level)
```

## Hybrid Connectivity

```
# Site-to-Site VPN — IPsec over internet; 2 tunnels/connection
#   (different AWS endpoints — configure BOTH); ~1.25 Gbps/tunnel
#   ceiling; BGP (dynamic) or static; Accelerated VPN option rides
#   Global Accelerator; minutes to deploy — "quick / temporary /
#   encrypted / backup" answer; ECMP over multiple tunnels via TGW
#   scales aggregate bandwidth
# Direct Connect — dedicated port at a DX location: 1/10/100/400 Gbps
#   dedicated (or 50 Mbps–25 Gbps hosted via partner); consistent
#   latency/bandwidth, cheaper per-GB egress than internet; weeks to
#   provision; NOT encrypted (MACsec or VPN-over-DX adds it —
#   cs aws-network-security)
# VIF types: private VIF → VGW/DX gateway (VPC access);
#   transit VIF → DX gateway → TGW; public VIF → AWS public endpoints
#   (S3, DynamoDB public IPs) — no VPC
# DX gateway — one DX connection → up to 20 VPCs across Regions
#   (via VGWs) or TGWs (via transit VIF)
# Resiliency: single DX = single point of failure. SLA tiers: dev =
#   DX + VPN backup (classic answer); high = 2 DX at 2 locations;
#   max = 2×2 (dual devices per location). BFD speeds failover.
# Route preference into VPC: most-specific prefix first; then DX
#   (private VIF) beats VPN — DX primary/VPN backup needs no tuning;
#   VPN-preferred requires more-specific routes over VPN (unusual)
```

## CloudFront vs Global Accelerator (Edge Duo)

```
# CloudFront — CDN: CACHES HTTP(S) content at 400+ POPs
#   Origins: S3 (lock down with Origin Access Control — bucket policy
#   allows only the distribution; legacy OAI), ALB/EC2 (custom origin;
#   restrict via CloudFront prefix list SG + custom origin header
#   checked at ALB/WAF), API GW, any HTTP server; origin groups =
#   origin failover
#   Behaviors: per-path routing (/api/* no-cache → ALB, /static/* long
#   TTL → S3); cache policies (keys: headers/cookies/query); TTLs +
#   invalidations (paid per path after 1000/mo) — version filenames
#   instead; compression; HTTP/3
#   Security: HTTPS enforcement, ACM cert in us-east-1, WAF, signed
#   URLs/cookies (paid/private content), geo restriction, field-level
#   encryption
#   Edge compute: CloudFront Functions (JS, µs, view-req/resp — header
#   rewrites, redirects) vs Lambda@Edge (Node/Python, ms, all 4 hooks,
#   origin-side logic — A/B, auth, origin selection)
#   VPC origins — reach PRIVATE ALB/NLB/EC2 without public exposure
# Global Accelerator — NO caching: 2 static anycast IPs, user enters
#   AWS backbone at nearest POP, routed to nearest healthy endpoint
#   (ALB/NLB/EC2/EIP) across Regions; instant failover (<30 s), client
#   IP preservation option
#   → TCP/UDP non-HTTP (gaming, VoIP, MQTT), "static IPs for allow-
#   lists", "deterministic multi-Region failover", HTTP apps needing
#   backbone but not caching
# Chooser: cacheable web content → CloudFront; static IP / UDP /
#   fastest Regional failover → Global Accelerator; both together is
#   valid (GA in front of ALBs + CloudFront for static)
```

## Load Balancer Selection (Recap Pointer)

```
# ALB — L7 routing (path/host/header), Lambda targets, Cognito auth,
#   WAF; no static IP (front with GA or NLB+ALB target)
# NLB — L4, millions rps, static IP per AZ, TLS passthrough,
#   PrivateLink backend
# GWLB — transparent appliance insertion (GENEVE)
# Full mechanics → cs aws-high-availability
```

## Hybrid & Multi-VPC DNS (Route 53 Resolver)

```
# VPC resolver (.2) answers: private hosted zones, VPC names, public
# On-prem → AWS names: INBOUND resolver endpoint (on-prem DNS
#   forwards zone → endpoint IPs in your subnets)
# AWS → on-prem names: OUTBOUND endpoint + forwarding rules
#   (corp.example.com → on-prem DNS IPs); share rules org-wide via RAM
# Private hosted zone across VPCs/accounts: associate VPCs to the zone
# "EC2 can't resolve corp hostnames" → outbound endpoint + rule
# Route 53 Profiles — apply zone/rule sets to many VPCs at once
```

## Data Transfer Cost Model (Task 4.4 Core)

```
# FREE: all ingress from internet; same-AZ private-IP traffic;
#   EC2→S3/DynamoDB same Region (via gateway endpoint / public path
#   in-Region); origin→CloudFront; S3→CloudFront→viewer S3-side
# PAID (typical us-east-1 magnitudes — memorize relations, not cents):
#   Cross-AZ within Region: ~0.01/GB EACH direction (0.02 round)
#   Cross-Region: 0.01–0.09+/GB depending on pair
#   Out to internet: ~0.09/GB first tiers (the big one)
#   NAT gateway: hourly + ~0.045/GB PROCESSING on top of transfer
#   Interface endpoints: hourly + ~0.01/GB processing
#   TGW: per-attachment hourly + ~0.02/GB processing
#   CloudFront egress: cheaper than EC2 egress + free origin fetch
# Cost-cutting reflexes:
#   S3/DynamoDB from private subnets → GATEWAY endpoints (free) not NAT
#   Heavy S3 API traffic through NAT = the classic bill horror →
#     gateway endpoint erases processing charges
#   Same-AZ placement for chatty tiers (or accept cross-AZ as HA tax)
#   Serve static via CloudFront, not straight from ALB/EC2
#   Compress everything; batch cross-Region replication choices
#   Heavy sustained on-prem transfer → DX per-GB beats internet rates
#   Keep replicas/consumers in the producer's Region when possible
#   Cross-AZ NAT: NAT-per-AZ also SAVES transfer (no cross-AZ hop)
# VPC endpoint break-even: endpoint hourly+processing vs NAT
#   processing on that traffic — high-volume AWS-API traffic always
#   wins with endpoints
```

## Scaling & Performance Extras

```
# ENI — secondary IPs/interfaces; ENI-based failover (move ENI to
#   standby instance keeps IP+MAC)
# Enhanced networking (ENA/SR-IOV) — default on Nitro, up to 200 Gbps
# EFA — OS-bypass fabric for MPI/HPC (cluster placement groups)
# Jumbo frames 9001 MTU inside VPC (1500 over internet/VPN; DX
#   supports jumbo); path MTU mismatches drop big packets
# Prefix lists — managed CIDR sets referenced in SGs/routes
# BYOIP — bring public ranges; IPAM tracks; public IPv4 costs hourly
#   now (another reason for IPv6 / fewer EIPs)
```

## Exam Traps

```
# Peering is NON-TRANSITIVE — any "A through B to C" wording → TGW
# Gateway endpoints: S3/DynamoDB only, free, unreachable from
#   on-prem/peer VPCs; interface endpoints work over DX/VPN
# NAT gateway is IPv4-only; IPv6 outbound-only = egress-only IGW
# NAT gateway lives in ONE AZ — per-AZ NAT for HA and lower cross-AZ
#   cost (cs aws-high-availability)
# Overlapping CIDRs kill peering AND VPN routing — plan with IPAM;
#   PrivateLink is the overlap-tolerant connectivity
# DX is not encrypted, not quick to provision, and needs redundancy
#   for SLA — three separate trap dimensions
# Two VPN tunnels exist for HA — configure both, or "single tunnel
#   down = outage" scenarios bite
# CloudFront cert must be in us-east-1; regional API/ALB certs local
# S3 static site direct = no HTTPS on custom domain → CloudFront+ACM
#   is the answer for "HTTPS on static site"
# Global Accelerator does NOT cache — "cache at edge" ≠ GA
# Client IP: ALB→XFF header, NLB preserves, CloudFront→
#   CloudFront-Viewer-Address; apps behind proxies must read headers
# "Restrict S3 origin to CloudFront only" → Origin Access Control
# Cross-AZ data transfer is billed BOTH directions — chatty
#   microservices across AZs add up; it is still the price of HA
# Interface endpoint DNS: private DNS ON or SDKs still hit public IPs
#   through NAT (silent cost leak)
```

## See Also

aws-saa-c03, aws-network-security, aws-high-availability, aws-storage, aws-cost-optimization, aws-monitoring-governance, vpc, subnetting, bgp, ipsec, dns, cloud-dns, nat, mtu, http3

## References

- [VPC User Guide](https://docs.aws.amazon.com/vpc/latest/userguide/what-is-amazon-vpc.html)
- [VPC endpoints & PrivateLink](https://docs.aws.amazon.com/vpc/latest/privatelink/what-is-privatelink.html)
- [Transit Gateway Guide](https://docs.aws.amazon.com/vpc/latest/tgw/what-is-transit-gateway.html)
- [Direct Connect User Guide](https://docs.aws.amazon.com/directconnect/latest/UserGuide/Welcome.html)
- [Direct Connect resiliency recommendations](https://aws.amazon.com/directconnect/resiliency-recommendation/)
- [CloudFront Developer Guide](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/Introduction.html)
- [Global Accelerator Developer Guide](https://docs.aws.amazon.com/global-accelerator/latest/dg/what-is-global-accelerator.html)
- [Route 53 Resolver endpoints](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/resolver.html)
- [AWS data transfer pricing](https://aws.amazon.com/ec2/pricing/on-demand/#Data_Transfer)
