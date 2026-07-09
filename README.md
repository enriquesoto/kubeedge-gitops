# kubeedge-gitops

GitOps repo for the KubeEdge-on-AKS project ([kubeedge](https://github.com/enriquesoto/kubeedge) — Terraform,
installs AKS + Argo CD). Argo CD reconciles everything below from this repo; no manual `kubectl`/`helm` needed
after the one-time bootstrap.

## Structure

- `argocd/bootstrap/root-application.yaml` — the one Application applied manually. Points Argo CD at `apps/`.
- `apps/` — flat directory of Argo CD `Application` manifests, one per app (app-of-apps pattern). Add a new
  app by dropping another file here.
- `charts/<app>/chart` — vendored upstream Helm chart. `charts/<app>/values.yaml` — our override values,
  referenced by the matching `apps/<app>.yaml` via `helm.valueFiles: [../values.yaml]`.

## Bootstrap (one-time, manual)

```bash
kubectl apply -f argocd/bootstrap/root-application.yaml
```

## Adding an app

1. Vendor its chart under `charts/<name>/chart`.
2. Add override values at `charts/<name>/values.yaml`.
3. Add `apps/<name>.yaml` (copy `apps/cloudcore.yaml` as a template).
4. Commit and push — Argo CD picks it up automatically.
