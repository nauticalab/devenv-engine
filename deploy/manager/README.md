# DevEnv Manager Deployment

This directory contains Kubernetes manifests for deploying the DevEnv Manager API server.

## Architecture

The DevEnv Manager provides an HTTP API for managing developer environment pods with authentication and authorization:

- **Authentication**: Kubernetes service account tokens (projected tokens with audience binding)
- **Authorization**: Developer-scoped access (users can only see/manage their own pods)
- **Zero-trust**: Every request is authenticated via TokenReview API

## Components

### Core Resources

- **Namespace** (`namespace.yaml`): `devenv-system` namespace for manager components
- **ServiceAccount** (`serviceaccount.yaml`):
  - `devenv-manager`: Service account for the manager deployment
  - `devenv-user-template`: Template for creating per-user service accounts
- **ClusterRole** (`clusterrole.yaml`):
  - `devenv-manager`: Permissions for manager (pod CRUD, TokenReview)
  - `devenv-user`: Documentation role for user permissions
- **ClusterRoleBinding** (`clusterrolebinding.yaml`): Binds manager SA to manager role
- **Deployment** (`deployment.yaml`): Manager deployment (2 replicas, health checks, security context)
- **Service** (`service.yaml`): ClusterIP service exposing manager on port 8080

### User Setup

Per-developer service accounts must be created following the naming pattern `devenv-{username}`:

```yaml
# Example: deploy/users/devenv-eywalker-sa.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: devenv-eywalker
  namespace: devenv-system
  labels:
    app.kubernetes.io/name: devenv-manager
    app.kubernetes.io/component: user
    developer: eywalker
```

## Deployment

### Prerequisites

- Kubernetes cluster (1.21+)
- kubectl configured
- Built manager container image

### Build and Push Image

```bash
# Build the manager binary
task build:manager

# Build container image
docker build -t ghcr.io/walkerlab/devenv-manager:latest -f deploy/manager/Dockerfile .

# Push to registry
docker push ghcr.io/walkerlab/devenv-manager:latest
```

### Deploy Manager

```bash
# Create namespace and RBAC
kubectl apply -f deploy/manager/namespace.yaml
kubectl apply -f deploy/manager/serviceaccount.yaml
kubectl apply -f deploy/manager/clusterrole.yaml
kubectl apply -f deploy/manager/clusterrolebinding.yaml

# Deploy manager
kubectl apply -f deploy/manager/deployment.yaml
kubectl apply -f deploy/manager/service.yaml

# Verify deployment
kubectl -n devenv-system get pods
kubectl -n devenv-system get svc
```

### Create User Service Accounts

```bash
# Create a service account for each developer
kubectl create serviceaccount devenv-eywalker -n devenv-system
kubectl label serviceaccount devenv-eywalker -n devenv-system developer=eywalker
```

## DevEnv Pod Configuration

Developer environment pods must be configured with:

1. **Projected service account token** for authentication
2. **Developer label** for authorization
3. **Manager URL** as environment variable

Example StatefulSet snippet:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: eywalker-devenv
  namespace: devenv
spec:
  template:
    metadata:
      labels:
        developer: eywalker # Required for authorization
    spec:
      serviceAccountName: devenv-eywalker # Must match pattern devenv-{username}

      containers:
        - name: devenv
          image: devenv:latest
          env:
            - name: DEVENV_MANAGER_URL
              value: "http://devenv-manager.devenv-system.svc.cluster.local:8080"

          volumeMounts:
            - name: devenv-manager-token
              mountPath: /var/run/secrets/tokens
              readOnly: true

      volumes:
        - name: devenv-manager-token
          projected:
            sources:
              - serviceAccountToken:
                  path: devenv-manager
                  expirationSeconds: 3600
                  audience: devenv-manager
```

## Usage

### From DevEnv Pod

```bash
# List your pods via manager
devenv pods list --remote --manager-url http://devenv-manager.devenv-system.svc.cluster.local:8080

# Delete a pod via manager
devenv pods delete my-pod --remote --manager-url http://devenv-manager.devenv-system.svc.cluster.local:8080 -n devenv
```

### Direct API Calls

```bash
# Get token from mounted volume
TOKEN=$(cat /var/run/secrets/tokens/devenv-manager)

# List pods
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Auth-Type: k8s-sa" \
     http://devenv-manager.devenv-system.svc.cluster.local:8080/api/v1/pods

# Health check (no auth required)
curl http://devenv-manager.devenv-system.svc.cluster.local:8080/api/v1/health
```

## Security Considerations

### Token Security

- **Audience-bound tokens**: Tokens are bound to `devenv-manager` audience
- **Short-lived**: Tokens expire after 1 hour (auto-rotated by kubelet)
- **Projected volumes**: Tokens are mounted read-only, not stored in secrets

### Authorization

- **Label-based filtering**: Manager filters pods by `developer` label
- **Service account naming**: Developer extracted from SA name pattern `devenv-{developer}`
- **Deletion validation**: Manager verifies pod ownership before deletion

### Network Security

- **ClusterIP service**: Manager not exposed outside cluster by default
- **TLS**: Consider adding TLS termination (Ingress, service mesh, or native HTTPS)
- **Network policies**: Restrict access to manager service if needed

## Monitoring

### Health Checks

```bash
# Check manager health
kubectl -n devenv-system exec -it devenv-manager-xxx -- \
  curl localhost:8080/api/v1/health

# Check version
kubectl -n devenv-system exec -it devenv-manager-xxx -- \
  curl localhost:8080/api/v1/version
```

### Logs

```bash
# View manager logs
kubectl -n devenv-system logs -f deployment/devenv-manager

# View logs for specific pod
kubectl -n devenv-system logs -f devenv-manager-xxx
```

## Troubleshooting

### Manager Pod Not Starting

```bash
# Check pod status
kubectl -n devenv-system get pods

# Check pod events
kubectl -n devenv-system describe pod devenv-manager-xxx

# Check logs
kubectl -n devenv-system logs devenv-manager-xxx
```

### Authentication Failures

1. Verify token is mounted:

   ```bash
   kubectl exec -it my-devenv-pod -- ls -la /var/run/secrets/tokens/
   ```

2. Check token contents (JWT):

   ```bash
   kubectl exec -it my-devenv-pod -- cat /var/run/secrets/tokens/devenv-manager | cut -d. -f2 | base64 -d
   ```

3. Verify service account exists:
   ```bash
   kubectl -n devenv-system get sa devenv-{username}
   ```

### Authorization Failures

1. Check pod labels:

   ```bash
   kubectl get pod my-pod -o jsonpath='{.metadata.labels.developer}'
   ```

2. Verify service account naming:

   ```bash
   # Should match pattern: devenv-{username}
   kubectl get pod my-devenv-pod -o jsonpath='{.spec.serviceAccountName}'
   ```

3. Check manager logs for auth errors:
   ```bash
   kubectl -n devenv-system logs -f deployment/devenv-manager | grep -i auth
   ```
