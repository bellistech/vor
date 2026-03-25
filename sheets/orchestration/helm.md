# Helm (Kubernetes Package Manager)

Template, package, and deploy Kubernetes applications using charts and releases.

## Repository Management

### Add and search repos

```bash
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo add jetstack https://charts.jetstack.io
helm repo update                            # fetch latest chart indexes
helm repo list
helm repo remove bitnami
```

### Search for charts

```bash
helm search repo nginx                      # search added repos
helm search repo bitnami/postgres --versions   # all available versions
helm search hub prometheus                  # search Artifact Hub
```

## Installing Charts

### Install a release

```bash
helm install my-nginx bitnami/nginx
helm install my-nginx bitnami/nginx --version 15.1.0
helm install my-nginx bitnami/nginx -n webstack --create-namespace
helm install my-nginx bitnami/nginx -f values-prod.yaml
helm install my-nginx bitnami/nginx --set replicaCount=3 --set service.type=LoadBalancer
helm install my-nginx bitnami/nginx --set-string annotations.app=web   # force string type
helm install my-nginx ./my-chart/           # from local directory
helm install my-nginx my-chart-1.0.0.tgz   # from local tarball
```

### Dry run and debug

```bash
helm install my-nginx bitnami/nginx --dry-run --debug   # render without applying
helm install my-nginx bitnami/nginx --dry-run --debug 2>&1 | less
```

## Upgrading Releases

```bash
helm upgrade my-nginx bitnami/nginx
helm upgrade my-nginx bitnami/nginx --version 15.2.0
helm upgrade my-nginx bitnami/nginx -f values-prod.yaml
helm upgrade my-nginx bitnami/nginx --set replicaCount=5
helm upgrade my-nginx bitnami/nginx --reuse-values --set image.tag=1.25
helm upgrade --install my-nginx bitnami/nginx   # install if not exists, upgrade if it does
helm upgrade my-nginx bitnami/nginx --atomic     # auto-rollback on failure
helm upgrade my-nginx bitnami/nginx --wait --timeout 5m
```

## Rollback

```bash
helm rollback my-nginx                      # rollback to previous revision
helm rollback my-nginx 3                    # rollback to revision 3
helm rollback my-nginx 3 --wait
```

## Listing and Inspecting

### List releases

```bash
helm list                                   # current namespace
helm list -A                                # all namespaces
helm list -n webstack                       # specific namespace
helm list --deployed                        # only deployed
helm list --failed                          # only failed
helm list --pending                         # only pending
helm list -o json                           # JSON output
```

### Release info

```bash
helm status my-nginx
helm history my-nginx                       # revision history
helm get values my-nginx                    # user-supplied values
helm get values my-nginx --all              # all values including defaults
helm get manifest my-nginx                  # rendered Kubernetes manifests
helm get notes my-nginx                     # post-install notes
helm get all my-nginx                       # everything
```

### Chart info

```bash
helm show chart bitnami/nginx               # Chart.yaml metadata
helm show values bitnami/nginx              # default values.yaml
helm show readme bitnami/nginx
helm show all bitnami/nginx
```

## Templating

### Render templates locally

```bash
helm template my-nginx bitnami/nginx                    # render all templates
helm template my-nginx bitnami/nginx -f values-prod.yaml
helm template my-nginx bitnami/nginx --set replicaCount=3
helm template my-nginx bitnami/nginx -s templates/deployment.yaml   # single template
helm template my-nginx ./my-chart/ --debug               # show debug info on errors
```

## Creating Charts

### Scaffold and develop

```bash
helm create my-chart                        # scaffold new chart
helm lint my-chart/                         # validate chart
helm lint my-chart/ --strict                # warnings are errors
helm lint my-chart/ -f values-prod.yaml     # lint with specific values
```

### Package and distribute

```bash
helm package my-chart/                      # creates my-chart-0.1.0.tgz
helm package my-chart/ --version 1.2.3      # override version
helm package my-chart/ -d ./releases/       # output directory
```

### Dependencies

```bash
helm dependency update my-chart/            # download dependencies from Chart.yaml
helm dependency list my-chart/
helm dependency build my-chart/             # rebuild charts/ directory
```

## Uninstalling

```bash
helm uninstall my-nginx
helm uninstall my-nginx -n webstack
helm uninstall my-nginx --keep-history      # keep release history for rollback
```

## Plugins

```bash
helm plugin list
helm plugin install https://github.com/databus23/helm-diff
helm plugin update diff
helm plugin uninstall diff
helm diff upgrade my-nginx bitnami/nginx -f values-prod.yaml   # using helm-diff
```

## Tips

- `helm upgrade --install` (or `-i`) is idempotent: use it in CI/CD pipelines instead of separate install/upgrade logic.
- `helm show values` is essential before installing a chart -- it shows every configurable option with documentation.
- `--atomic` on upgrade automatically rolls back if the deploy fails or times out. Always use this in production.
- `helm template` renders locally without a cluster. Use it to inspect what will be applied or to pipe into `kubectl diff`.
- `--reuse-values` on upgrade keeps your previous values and applies only the new `--set` overrides. Without it, values reset to defaults.
- `helm get values my-release` shows what you customized. `--all` shows the full merged values.
- `helm diff` plugin is invaluable: it shows a colored diff of what `helm upgrade` would change before you apply.
- Chart version (Chart.yaml `version`) and app version (`appVersion`) are different. Chart version tracks the Helm chart itself; app version tracks the software inside.
- Use `helm list --failed` to find botched releases that need cleanup.
- `helm uninstall --keep-history` lets you inspect or rollback a deleted release; without it, the history is gone.

## References

- [Helm Documentation](https://helm.sh/docs/)
- [Helm CLI Reference](https://helm.sh/docs/helm/)
- [Chart Template Guide](https://helm.sh/docs/chart_template_guide/)
- [Chart.yaml Schema](https://helm.sh/docs/topics/charts/)
- [Helm Built-in Objects](https://helm.sh/docs/chart_template_guide/builtin_objects/)
- [Helm Best Practices](https://helm.sh/docs/chart_best_practices/)
- [Helm GitHub Repository](https://github.com/helm/helm)
- [Artifact Hub — Helm Chart Registry](https://artifacthub.io/)
- [Helm Provenance and Integrity](https://helm.sh/docs/topics/provenance/)
- [Library Charts](https://helm.sh/docs/topics/library_charts/)
