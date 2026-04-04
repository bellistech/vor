# Cilium (eBPF-Based Networking and Security)

Cilium is an eBPF-powered CNI plugin providing L3/L4/L7 network policies, transparent encryption, Hubble observability, sidecar-free service mesh, cluster mesh for multi-cluster connectivity, BGP peering, bandwidth management, and Tetragon runtime security enforcement for Kubernetes.

## Installation

### Install Cilium CLI

```bash
# Install cilium CLI
CILIUM_CLI_VERSION=$(curl -s https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt)
curl -L --fail \
  https://github.com/cilium/cilium-cli/releases/download/${CILIUM_CLI_VERSION}/cilium-linux-amd64.tar.gz \
  | sudo tar xz -C /usr/local/bin

# Install via Homebrew
brew install cilium-cli

# Verify
cilium version
```

### Deploy Cilium

```bash
# Install Cilium on Kubernetes
cilium install

# Install with specific options
cilium install \
  --version 1.16.0 \
  --set kubeProxyReplacement=true \
  --set hubble.relay.enabled=true \
  --set hubble.ui.enabled=true

# Install via Helm
helm repo add cilium https://helm.cilium.io/
helm install cilium cilium/cilium \
  --namespace kube-system \
  --set hubble.relay.enabled=true \
  --set hubble.ui.enabled=true \
  --set kubeProxyReplacement=true \
  --set bpf.masquerade=true

# Validate installation
cilium status --wait
cilium connectivity test
```

## Network Policies (L3/L4)

### Basic L3/L4 Policies

```yaml
# Default deny all ingress and egress
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: default-deny
spec:
  endpointSelector: {}
  ingress:
    - {}
  egress:
    - {}
---
# Allow specific pod-to-pod communication (L3/L4)
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: allow-frontend-to-api
spec:
  endpointSelector:
    matchLabels:
      app: api
  ingress:
    - fromEndpoints:
        - matchLabels:
            app: frontend
      toPorts:
        - ports:
            - port: "8080"
              protocol: TCP
```

## Network Policies (L7)

### HTTP-Aware Policies

```yaml
# L7 HTTP policy (filter by path/method/headers)
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: api-l7-rules
spec:
  endpointSelector:
    matchLabels:
      app: api
  ingress:
    - fromEndpoints:
        - matchLabels:
            app: frontend
      toPorts:
        - ports:
            - port: "8080"
              protocol: TCP
          rules:
            http:
              - method: GET
                path: "/api/v1/public/.*"
              - method: POST
                path: "/api/v1/data"
                headers:
                  - 'Content-Type: application/json'
              - method: GET
                path: "/healthz"
```

## Hubble Observability

### Install and Use Hubble

```bash
# Enable Hubble
cilium hubble enable --ui

# Install Hubble CLI
HUBBLE_VERSION=$(curl -s https://raw.githubusercontent.com/cilium/hubble/master/stable.txt)
curl -L --fail \
  https://github.com/cilium/hubble/releases/download/${HUBBLE_VERSION}/hubble-linux-amd64.tar.gz \
  | sudo tar xz -C /usr/local/bin

# Port-forward Hubble relay
cilium hubble port-forward &

# Observe all flows
hubble observe

# Filter by namespace
hubble observe --namespace production

# Filter by pod
hubble observe --pod production/frontend-abc123

# Filter by verdict (allowed/dropped)
hubble observe --verdict DROPPED
hubble observe --verdict FORWARDED

# Filter by L7 protocol
hubble observe --protocol http
hubble observe --http-status-code 500

# Filter by destination
hubble observe --to-pod production/api
hubble observe --to-fqdn api.example.com

# JSON output
hubble observe --output json

# Hubble UI (browser)
cilium hubble ui
# Opens http://localhost:12000

# Flow metrics
hubble observe --print-raw-filters
```

## Service Mesh (Sidecar-Free)

### Cilium Service Mesh

```bash
# Enable service mesh features
helm upgrade cilium cilium/cilium \
  --namespace kube-system \
  --set ingressController.enabled=true \
  --set envoyConfig.enabled=true \
  --set loadBalancer.l7.backend=envoy

# Cilium Ingress
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: myapp-ingress
  annotations:
    ingress.cilium.io/loadbalancer-mode: shared
spec:
  ingressClassName: cilium
  rules:
    - host: myapp.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: myapp
                port:
                  number: 8080
EOF

# Enable mutual TLS (mTLS) via SPIRE
helm upgrade cilium cilium/cilium \
  --namespace kube-system \
  --set authentication.mutual.spire.enabled=true \
  --set authentication.mutual.spire.install.enabled=true
```

## Cluster Mesh

### Multi-Cluster Connectivity

```bash
# Enable cluster mesh
cilium clustermesh enable --service-type LoadBalancer

# Connect two clusters
cilium clustermesh connect --destination-context cluster2

# Check cluster mesh status
cilium clustermesh status

# Global services (available across clusters)
kubectl annotate service myapp \
  io.cilium/global-service="true"

# Shared services with affinity (prefer local)
kubectl annotate service myapp \
  io.cilium/global-service="true" \
  io.cilium/shared-service="true" \
  io.cilium/service-affinity="local"
```

## BGP and Bandwidth Management

### BGP Peering

```yaml
# CiliumBGPPeeringPolicy
apiVersion: cilium.io/v2alpha1
kind: CiliumBGPPeeringPolicy
metadata:
  name: bgp-peering
spec:
  nodeSelector:
    matchLabels:
      bgp-policy: rack1
  virtualRouters:
    - localASN: 65001
      exportPodCIDR: true
      neighbors:
        - peerAddress: 10.0.0.1/32
          peerASN: 65000
          connectRetryTimeSeconds: 120
          holdTimeSeconds: 90
          keepAliveTimeSeconds: 30
```

### Bandwidth Manager

```bash
# Enable bandwidth manager
helm upgrade cilium cilium/cilium \
  --namespace kube-system \
  --set bandwidthManager.enabled=true \
  --set bandwidthManager.bbr=true

# Apply bandwidth annotations to pods
kubectl annotate pod myapp-pod \
  kubernetes.io/egress-bandwidth="10M" \
  kubernetes.io/ingress-bandwidth="20M"
```

## Tetragon (Runtime Security)

### Runtime Enforcement

```bash
# Install Tetragon
helm install tetragon cilium/tetragon \
  --namespace kube-system

# Install tetra CLI
curl -L https://github.com/cilium/tetragon/releases/latest/download/tetra-linux-amd64.tar.gz \
  | sudo tar xz -C /usr/local/bin

# Observe process events
kubectl exec -n kube-system ds/tetragon -c tetragon -- tetra getevents

# Watch process execution
kubectl exec -n kube-system ds/tetragon -c tetragon -- \
  tetra getevents --process myapp

# TracingPolicy: block execution of specific binaries
kubectl apply -f - <<EOF
apiVersion: cilium.io/v1alpha1
kind: TracingPolicy
metadata:
  name: block-curl-wget
spec:
  kprobes:
    - call: sys_execve
      syscall: true
      args:
        - index: 0
          type: string
      selectors:
        - matchArgs:
            - index: 0
              operator: In
              values:
                - /usr/bin/curl
                - /usr/bin/wget
          matchActions:
            - action: Sigkill
EOF
```

## Tips

- Enable `kubeProxyReplacement=true` to replace kube-proxy entirely with eBPF for better performance
- Always start with default-deny CiliumNetworkPolicy and explicitly allow required traffic flows
- Use L7 HTTP policies to restrict API access by method and path, not just IP and port
- Deploy Hubble with the UI for real-time service dependency maps and flow visibility
- Cilium's sidecar-free service mesh uses eBPF and shared Envoy instances, reducing pod overhead
- Use DNS-aware policies (`toFQDNs`) for egress to external services instead of hardcoding IP CIDRs
- Tetragon TracingPolicies can kill processes in real-time; test thoroughly in staging before production
- Enable bandwidth manager with BBR congestion control for fair bandwidth sharing across pods
- Cluster mesh global services enable multi-cluster failover without external load balancers
- Use `cilium connectivity test` after every upgrade to verify policy enforcement is working
- Monitor `hubble observe --verdict DROPPED` to detect policy misconfiguration quickly
- BGP peering eliminates the need for MetalLB in bare-metal Kubernetes deployments

## See Also

- networking, kubernetes, envoy, istio, linkerd, falco, seccomp, ebpf

## References

- [Cilium Documentation](https://docs.cilium.io/)
- [Hubble Documentation](https://docs.cilium.io/en/stable/observability/hubble/)
- [Tetragon Documentation](https://tetragon.io/docs/)
- [Cilium Network Policies](https://docs.cilium.io/en/stable/security/policy/)
- [Cilium Cluster Mesh](https://docs.cilium.io/en/stable/network/clustermesh/)
- [eBPF.io](https://ebpf.io/)
- [Cilium Service Mesh](https://docs.cilium.io/en/stable/network/servicemesh/)
