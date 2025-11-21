# DevENV Manager Integration

This document describes the integration of the DevENV Manager API into the existing devenv generate system.

## Changes Made

### 1. Service Account Template

Created `internal/templates/template_files/dev/manifests/serviceaccount.tmpl`:

- Generates a Kubernetes ServiceAccount for each developer
- Naming pattern: `devenv-{username}` (e.g., `devenv-eywalker`)
- Includes `developer` label for identification
- Automatically created when running `devenv generate`

### 2. Updated StatefulSet Template

Modified `internal/templates/template_files/dev/manifests/statefulset.tmpl`:

**Pod Labels:**

- Added `developer: {{.Name}}` label to pod template
- Required for manager API authorization (filters pods by developer)

**Service Account:**

- Non-admin users now use `devenv-{username}` service account
- Admin users continue to use `k8s-launcher` service account

**Environment Variables:**

- Added `DEVENV_MANAGER_URL`: Points to manager service
- Added `DEVELOPER`: Developer name for convenience

**Projected Token Volume:**

- Added `devenv-manager-token` volume mount at `/var/run/secrets/tokens`
- Token configuration:
  - Path: `devenv-manager` (full path: `/var/run/secrets/tokens/devenv-manager`)
  - Audience: `devenv-manager` (matches manager's expected audience)
  - Expiration: 3600 seconds (1 hour, auto-rotated by kubelet)

### 3. Template Renderer Update

Modified `internal/templates/renderer.go`:

- Added `serviceaccount` to `devTemplatesToRender` list
- Service account manifest is now generated first (before statefulset)

### 4. Generate Command Enhancement

Modified `cmd/devenv/generate.go`:

- Added `parseDeveloperInput()` function
- Now accepts multiple input formats:
  - Developer name only: `devenv generate eywalker`
  - Directory path: `devenv generate developers/eywalker`
  - Full config path: `devenv generate developers/eywalker/devenv-config.yaml`
- Automatically extracts developer name from any format

## Usage

### Generate Manifests for Single Developer

```bash
# Any of these work:
devenv generate eywalker
devenv generate developers/eywalker
devenv generate developers/eywalker/devenv-config.yaml
```

Generated files in `build/eywalker/`:

- `serviceaccount.yaml` - Service account with projected token support
- `statefulset.yaml` - Pod with manager token volume and developer label
- `service.yaml` - Service definition
- `env-vars.yaml` - Environment variables ConfigMap
- `startup-scripts.yaml` - Startup scripts ConfigMap
- `ingress.yaml` - Ingress configuration (if applicable)

### Generate for All Developers

```bash
devenv generate --all-developers
```

### Deploy Manager First

Before using the devenv pods with remote access:

```bash
# Deploy the manager
kubectl apply -f deploy/manager/namespace.yaml
kubectl apply -f deploy/manager/serviceaccount.yaml
kubectl apply -f deploy/manager/clusterrole.yaml
kubectl apply -f deploy/manager/clusterrolebinding.yaml
kubectl apply -f deploy/manager/deployment.yaml
kubectl apply -f deploy/manager/service.yaml

# Or use kustomize
kubectl apply -k deploy/manager/
```

### Deploy Developer Environment

```bash
# Generate manifests
devenv generate eywalker

# Apply to cluster
kubectl apply -f build/eywalker/
```

## What Developers Get

Each developer environment pod now has:

1. **Service Account**: `devenv-{username}` with projected token
2. **Manager Access**: Environment variable `DEVENV_MANAGER_URL` pre-configured
3. **Token Volume**: Mounted at `/var/run/secrets/tokens/devenv-manager`
4. **Developer Label**: Pod labeled with `developer: {username}`

## Using Remote Pod Management

From within a devenv pod:

```bash
# List your pods via manager
devenv pods list --remote --manager-url $DEVENV_MANAGER_URL

# Delete a pod via manager
devenv pods delete my-pod --remote --manager-url $DEVENV_MANAGER_URL -n devenv

# Or use the default manager URL (set in env)
devenv pods list --remote
```

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                       │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ devenv-system namespace                               │  │
│  │                                                        │  │
│  │  ┌──────────────────────────────────────────────┐   │  │
│  │  │ DevENV Manager Deployment                     │   │  │
│  │  │  - HTTP API (port 8080)                       │   │  │
│  │  │  - TokenReview authentication                 │   │  │
│  │  │  - Developer-scoped authorization             │   │  │
│  │  └──────────────────────────────────────────────┘   │  │
│  │                                                        │  │
│  │  ClusterRole: devenv-manager                          │  │
│  │    - pods: get, list, delete                          │  │
│  │    - tokenreviews: create                             │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ devenv namespace                                      │  │
│  │                                                        │  │
│  │  ┌────────────────────────────────────────────┐     │  │
│  │  │ Developer Pod (eywalker)                    │     │  │
│  │  │  ServiceAccount: devenv-eywalker            │     │  │
│  │  │  Labels: developer=eywalker                 │     │  │
│  │  │                                              │     │  │
│  │  │  Volumes:                                    │     │  │
│  │  │    /var/run/secrets/tokens/devenv-manager   │     │  │
│  │  │      (projected SA token, audience-bound)   │     │  │
│  │  │                                              │     │  │
│  │  │  Environment:                                │     │  │
│  │  │    DEVENV_MANAGER_URL=http://...            │     │  │
│  │  │    DEVELOPER=eywalker                        │     │  │
│  │  └────────────────────────────────────────────┘     │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘

Flow:
1. devenv CLI reads token from /var/run/secrets/tokens/devenv-manager
2. Makes HTTP request to manager with Bearer token + X-Auth-Type: k8s-sa
3. Manager validates token via TokenReview API
4. Manager extracts developer name from SA (devenv-{username})
5. Manager filters/manages only pods with matching developer label
```

## Security Model

### Authentication

- **Projected service account tokens** (not legacy tokens)
- **Audience-bound**: Token only valid for `devenv-manager` audience
- **Short-lived**: 1-hour expiration, auto-rotated by kubelet
- **Read-only mount**: Token file cannot be modified

### Authorization

- **Label-based filtering**: Manager only shows pods with matching developer label
- **Service account naming**: Developer extracted from `devenv-{username}` pattern
- **Deletion validation**: Manager verifies pod ownership before deletion
- **Zero-trust**: Every request re-authenticated via TokenReview

## Backward Compatibility

- Existing devenv pods without manager integration continue to work
- Admin users with `k8s-launcher` SA are unchanged
- Direct kubectl access is not affected
- Manager integration is opt-in (use `--remote` flag)

## Future Enhancements

Potential additions to consider:

- GitHub OAuth integration (alternative auth provider)
- OIDC authentication support
- Pod creation via manager API
- Pod logs streaming
- Pod exec/attach via manager
- Resource usage metrics
- Audit logging
