# DevEnv Manager Troubleshooting Guide

This guide covers common issues and solutions when working with the DevEnv Manager API.

## Table of Contents

- [Authentication Issues](#authentication-issues)
- [Authorization Issues](#authorization-issues)
- [Pod Management Issues](#pod-management-issues)
- [Deployment Issues](#deployment-issues)
- [Configuration Issues](#configuration-issues)

## Authentication Issues

### Problem: "authentication failed" or "token not authenticated"

**Symptoms:**

- API requests return 401 Unauthorized
- Manager logs show "token not authenticated"

**Possible Causes:**

1. **Token file not mounted**

   ```bash
   # Check if token volume is mounted in pod
   kubectl describe pod <pod-name> -n devenv | grep -A5 "projected-token"

   # Verify token file exists
   kubectl exec <pod-name> -n devenv -- ls -la /var/run/secrets/tokens/
   ```

   **Solution:** Ensure StatefulSet has projected token volume:

   ```yaml
   volumes:
     - name: projected-token
       projected:
         sources:
           - serviceAccountToken:
               audience: devenv-manager
               expirationSeconds: 3600
               path: devenv-manager
   ```

2. **Wrong audience**

   ```bash
   # Check token audience
   kubectl exec <pod-name> -n devenv -- cat /var/run/secrets/tokens/devenv-manager
   ```

   **Solution:** Manager expects audience `devenv-manager`. Verify manager startup flags:

   ```bash
   kubectl logs -n devenv deployment/devenv-manager | grep audience
   ```

3. **Expired token**

   Tokens expire after 3600 seconds (1 hour) by default. Kubernetes automatically rotates them.

   **Solution:** Wait a few minutes for automatic rotation, or restart the pod:

   ```bash
   kubectl delete pod <pod-name> -n devenv
   ```

### Problem: "invalid service account username"

**Symptoms:**

- Authentication succeeds but returns "invalid service account username"
- Manager logs show username parsing errors

**Possible Causes:**

ServiceAccount not following naming convention `devenv-{username}`.

**Solution:**

```bash
# Check ServiceAccount name
kubectl get sa -n devenv

# Verify pod is using correct ServiceAccount
kubectl get pod <pod-name> -n devenv -o jsonpath='{.spec.serviceAccountName}'
```

Should be: `devenv-<username>` (e.g., `devenv-eywalker`)

## Authorization Issues

### Problem: "You can only access your own pods" or 403 Forbidden

**Symptoms:**

- Can list/delete some pods but not others
- Manager logs show "pod developer mismatch"

**Explanation:**

DevEnv Manager enforces developer-scoped authorization. Users can only access pods with `developer={username}` label.

**Verification:**

```bash
# Check pod labels
kubectl get pod <pod-name> -n devenv --show-labels

# Should see: developer=<username>
```

**Solution:**

Ensure pods have correct label:

```yaml
metadata:
  labels:
    developer: eywalker # Must match ServiceAccount developer name
```

### Problem: "No developer identity found"

**Symptoms:**

- API returns "No developer identity found" or 401

**Possible Causes:**

1. Missing authentication header
2. Token not properly extracted from ServiceAccount

**Solution:**

When using `devenv` CLI with `--remote` flag, verify token is accessible:

```bash
# From inside pod
cat /var/run/secrets/tokens/devenv-manager

# Check DEVENV_MANAGER_URL environment variable
echo $DEVENV_MANAGER_URL
```

## Pod Management Issues

### Problem: "Pod not found" when deleting

**Symptoms:**

- DELETE request returns 404
- Pod exists in Kubernetes

**Possible Causes:**

1. **Wrong namespace**

   ```bash
   # Check which namespace pod is in
   kubectl get pod <pod-name> --all-namespaces
   ```

2. **Pod doesn't belong to authenticated user**

   Manager filters pods by developer label before deletion.

**Solution:**

Use correct namespace and verify ownership:

```bash
kubectl get pod <pod-name> -n devenv -L developer
```

### Problem: Cannot list any pods

**Symptoms:**

- API returns empty list
- Pods exist in cluster

**Possible Causes:**

1. **Pods in different namespace**

   ```bash
   # List pods with developer label
   kubectl get pods -n devenv -l developer=<username>
   ```

2. **Missing developer label**

**Solution:**

Verify StatefulSet template includes developer label:

```yaml
spec:
  template:
    metadata:
      labels:
        developer: { { .Name } }
```

## Deployment Issues

### Problem: Manager deployment failing

**Symptoms:**

- Manager pods in CrashLoopBackOff
- Deployment not ready

**Diagnosis:**

```bash
# Check pod status
kubectl get pods -n devenv -l app=devenv-manager

# View logs
kubectl logs -n devenv deployment/devenv-manager

# Check events
kubectl describe deployment -n devenv devenv-manager
```

**Common Issues:**

1. **RBAC permissions missing**

   ```bash
   # Verify ClusterRole
   kubectl get clusterrole devenv-manager-role -o yaml

   # Verify ClusterRoleBinding
   kubectl get clusterrolebinding devenv-manager-binding -o yaml
   ```

   Should have permissions for:

   - `pods` (get, list, delete)
   - `tokenreviews` (create)

2. **Port conflicts**

   Check if port 8080 is available:

   ```bash
   kubectl get svc -n devenv devenv-manager
   ```

### Problem: Manager not accessible from pods

**Symptoms:**

- `devenv pods list --remote` fails with connection error
- Cannot reach http://devenv-manager:8080

**Diagnosis:**

```bash
# From inside a devenv pod, test connectivity
kubectl exec -it <pod-name> -n devenv -- curl http://devenv-manager:8080/api/v1/health

# Check service DNS
kubectl exec -it <pod-name> -n devenv -- nslookup devenv-manager
```

**Solutions:**

1. **Service not created**

   ```bash
   kubectl get svc -n devenv devenv-manager
   ```

2. **Wrong namespace**

   Manager and pods must be in same namespace (`devenv`):

   ```bash
   kubectl get all -n devenv
   ```

3. **Service selector mismatch**
   ```bash
   kubectl get svc -n devenv devenv-manager -o yaml | grep -A5 selector
   kubectl get pods -n devenv -l app=devenv-manager --show-labels
   ```

## Configuration Issues

### Problem: Manager URL not configured correctly

**Symptoms:**

- `--remote` flag doesn't work
- CLI tries to connect to wrong URL

**Solution:**

Set `managerURL` in config:

```yaml
# In devenv-config.yaml or global-config.yaml
managerURL: http://devenv-manager:8080
```

Or use environment variable in pod:

```bash
export DEVENV_MANAGER_URL=http://devenv-manager:8080
```

### Problem: Service account not generated

**Symptoms:**

- No ServiceAccount created after `devenv generate`
- Pod fails to start with "serviceaccount not found"

**Diagnosis:**

```bash
# Check if serviceaccount.yaml was generated
ls output/<username>/dev/manifests/serviceaccount.yaml

# Apply it manually if missing
kubectl apply -f output/<username>/dev/manifests/serviceaccount.yaml
```

**Solution:**

Ensure `devenv generate` runs successfully:

```bash
devenv generate <username> --output ./output
```

Check generated files include `serviceaccount.yaml`.

## Debugging Tips

### Enable verbose logging

Manager:

```bash
# Edit deployment to add verbose flag (if supported)
kubectl edit deployment -n devenv devenv-manager
```

DevEnv CLI:

```bash
# Use with increased verbosity (if supported)
devenv pods list --remote -v
```

### Check TokenReview API

Verify TokenReview is working:

```bash
# From inside manager pod
kubectl exec -it <manager-pod> -n devenv -- sh

# Try creating a TokenReview manually
cat <<EOF | kubectl create -f -
apiVersion: authentication.k8s.io/v1
kind: TokenReview
spec:
  token: "<paste-token-here>"
  audiences:
    - devenv-manager
EOF
```

### Monitor manager logs

```bash
# Follow logs in real-time
kubectl logs -n devenv deployment/devenv-manager -f

# Get logs from all replicas
kubectl logs -n devenv -l app=devenv-manager --all-containers=true
```

### Verify RBAC

```bash
# Check if manager SA can perform operations
kubectl auth can-i list pods --as=system:serviceaccount:devenv:devenv-manager -n devenv
kubectl auth can-i delete pods --as=system:serviceaccount:devenv:devenv-manager -n devenv
kubectl auth can-i create tokenreviews --as=system:serviceaccount:devenv:devenv-manager
```

## Getting Help

If you're still experiencing issues:

1. Collect diagnostics:

   ```bash
   kubectl get all -n devenv -o yaml > devenv-diagnostics.yaml
   kubectl logs -n devenv deployment/devenv-manager --tail=100 > manager-logs.txt
   ```

2. Check the main README for architecture details
3. Review [deploy/manager/README.md](../deploy/manager/README.md) for deployment specifics
4. Search existing GitHub issues
5. Open a new issue with diagnostics attached
