# Kubernetes (Container Orchestration Platform)

Deploy, scale, and manage containerized applications across clusters using `kubectl`.

## Context and Configuration

### Manage clusters and contexts

```bash
kubectl config get-contexts
kubectl config current-context
kubectl config use-context production
kubectl config set-context --current --namespace=myapp
kubectl config view --minify                # show current context config only
kubectl cluster-info
```

## Namespaces

```bash
kubectl get namespaces
kubectl create namespace staging
kubectl delete namespace staging
kubectl get all -n kube-system              # resources in specific namespace
kubectl get pods --all-namespaces           # across all namespaces
kubectl get pods -A                         # shorthand for --all-namespaces
```

## Pods

### List and inspect

```bash
kubectl get pods
kubectl get pods -o wide                    # show node and IP
kubectl get pods -o yaml                    # full YAML output
kubectl get pods -l app=web                 # filter by label
kubectl get pods --field-selector status.phase=Running
kubectl describe pod web-abc123             # events, conditions, volumes
```

### Create and delete

```bash
kubectl run debug --image=alpine --rm -it -- sh              # ephemeral debug pod
kubectl run nginx --image=nginx:alpine --port=80
kubectl delete pod web-abc123
kubectl delete pod web-abc123 --grace-period=0 --force       # immediate delete
kubectl delete pods -l app=web                                # by label
```

### Logs

```bash
kubectl logs web-abc123
kubectl logs -f web-abc123                  # follow
kubectl logs --tail=100 web-abc123
kubectl logs web-abc123 -c sidecar         # specific container in multi-container pod
kubectl logs -l app=web --all-containers   # all pods matching label
kubectl logs --previous web-abc123          # logs from crashed/restarted container
kubectl logs --since=1h web-abc123
```

### Exec

```bash
kubectl exec -it web-abc123 -- sh
kubectl exec web-abc123 -- cat /etc/config/app.yaml
kubectl exec -it web-abc123 -c sidecar -- bash   # specific container
```

### Port forwarding

```bash
kubectl port-forward pod/web-abc123 8080:80
kubectl port-forward svc/web 8080:80           # forward to service
kubectl port-forward deploy/web 8080:80        # forward to deployment
```

## Deployments

### Manage deployments

```bash
kubectl get deployments
kubectl get deploy                          # shorthand
kubectl describe deploy web
kubectl create deployment web --image=nginx:alpine --replicas=3
kubectl apply -f deployment.yaml
kubectl delete deployment web
```

### Scaling

```bash
kubectl scale deployment web --replicas=5
kubectl autoscale deployment web --min=2 --max=10 --cpu-percent=80
kubectl get hpa                             # horizontal pod autoscaler
```

### Rollouts

```bash
kubectl rollout status deployment/web
kubectl rollout history deployment/web
kubectl rollout undo deployment/web                    # rollback to previous
kubectl rollout undo deployment/web --to-revision=3    # rollback to specific
kubectl rollout restart deployment/web                 # rolling restart
kubectl rollout pause deployment/web
kubectl rollout resume deployment/web
```

### Update image

```bash
kubectl set image deployment/web web=nginx:1.25-alpine
```

## Services

```bash
kubectl get services
kubectl get svc
kubectl describe svc web
kubectl expose deployment web --port=80 --target-port=8080 --type=ClusterIP
kubectl expose deployment web --port=80 --type=LoadBalancer
kubectl expose deployment web --port=80 --type=NodePort
kubectl delete svc web
```

## ConfigMaps and Secrets

### ConfigMaps

```bash
kubectl create configmap app-config --from-literal=DB_HOST=postgres --from-literal=DB_PORT=5432
kubectl create configmap app-config --from-file=config.yaml
kubectl create configmap app-config --from-env-file=.env
kubectl get configmaps
kubectl describe configmap app-config
kubectl get configmap app-config -o yaml
kubectl delete configmap app-config
```

### Secrets

```bash
kubectl create secret generic db-creds --from-literal=password=supersecret
kubectl create secret generic tls-cert --from-file=tls.crt --from-file=tls.key
kubectl create secret docker-registry regcred --docker-server=registry.io --docker-username=user --docker-password=pass
kubectl get secrets
kubectl get secret db-creds -o jsonpath='{.data.password}' | base64 -d
kubectl delete secret db-creds
```

## Apply and Delete

```bash
kubectl apply -f manifest.yaml
kubectl apply -f ./manifests/                # apply all files in directory
kubectl apply -k ./overlays/production/      # kustomize
kubectl delete -f manifest.yaml
kubectl diff -f manifest.yaml               # preview changes before apply
```

## Resource Inspection

### Get with formatting

```bash
kubectl get pods -o json
kubectl get pods -o jsonpath='{.items[*].metadata.name}'
kubectl get pods -o custom-columns='NAME:.metadata.name,STATUS:.status.phase'
kubectl get pods --sort-by=.metadata.creationTimestamp
kubectl get events --sort-by=.lastTimestamp
```

### Top (resource usage)

```bash
kubectl top nodes
kubectl top pods
kubectl top pods --containers              # per-container usage
kubectl top pods -l app=web
```

## Node Management

### Drain and cordon

```bash
kubectl get nodes
kubectl describe node worker-1
kubectl cordon worker-1                    # mark unschedulable
kubectl uncordon worker-1
kubectl drain worker-1 --ignore-daemonsets --delete-emptydir-data
```

### Taints and tolerations

```bash
kubectl taint nodes worker-1 gpu=true:NoSchedule
kubectl taint nodes worker-1 gpu=true:NoSchedule-    # remove taint (trailing dash)
kubectl describe node worker-1 | grep Taints
```

### Labels

```bash
kubectl label nodes worker-1 disktype=ssd
kubectl label nodes worker-1 disktype-                # remove label
kubectl get nodes -l disktype=ssd
kubectl label pods web-abc123 env=production
```

## Copy Files

```bash
kubectl cp web-abc123:/var/log/app.log ./app.log
kubectl cp ./config.yaml web-abc123:/etc/app/config.yaml
kubectl cp web-abc123:/data ./backup -c sidecar   # from specific container
```

## Tips

- `kubectl get` supports short names: `po` (pods), `svc` (services), `deploy` (deployments), `ns` (namespaces), `cm` (configmaps), `no` (nodes).
- Always use `kubectl diff -f` before `kubectl apply -f` in production to preview changes.
- `kubectl describe` shows events at the bottom, which is usually where you find the reason for failures.
- `--dry-run=client -o yaml` generates YAML without applying: `kubectl create deployment web --image=nginx --dry-run=client -o yaml > deploy.yaml`.
- Secrets are base64-encoded, not encrypted. Use external tools (Sealed Secrets, SOPS, Vault) for real secret management.
- `kubectl rollout undo` is the fastest way to recover from a bad deployment.
- `kubectl port-forward` works with pods, services, and deployments. Service-level forwarding respects load balancing.
- Set a default namespace with `kubectl config set-context --current --namespace=myapp` to avoid typing `-n` everywhere.
- `kubectl run --rm -it debug --image=alpine -- sh` is indispensable for in-cluster network debugging.

## See Also

- helm
- docker
- containerd
- podman
- terraform
- prometheus

## References

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [kubectl Reference](https://kubernetes.io/docs/reference/kubectl/)
- [Kubernetes API Reference](https://kubernetes.io/docs/reference/kubernetes-api/)
- [kubectl Cheat Sheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/)
- [Kubernetes Concepts](https://kubernetes.io/docs/concepts/)
- [Managing Resources](https://kubernetes.io/docs/concepts/cluster-administration/manage-deployment/)
- [Kubernetes Networking Model](https://kubernetes.io/docs/concepts/services-networking/)
- [RBAC Authorization](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- [ConfigMaps and Secrets](https://kubernetes.io/docs/concepts/configuration/)
- [Kubernetes GitHub Repository](https://github.com/kubernetes/kubernetes)
- [Kubernetes the Hard Way](https://github.com/kelseyhightower/kubernetes-the-hard-way)
