# Manager API Server Implementation Plan

**Status:** In Progress  
**Started:** November 19, 2025  
**Goal:** Add HTTP API server to manager with K8s SA token authentication and client support in devenv CLI

---

## Overview

Transform the manager into a dual-mode tool (CLI + HTTP server) that provides REST API endpoints for managing developer environment pods. Authentication via projected Kubernetes service account tokens with zero-trust architecture.

---

## Architecture

### Authentication Flow

```
┌──────────────────┐                  ┌─────────────┐                ┌──────────┐
│ devenv CLI       │                  │   manager   │                │ K8s API  │
│ (in eywalker pod)│                  │   server    │                │          │
│                  │                  │             │                │          │
│ 1. Read SA token │                  │             │                │          │
│    from volume   │                  │             │                │          │
│                  │                  │             │                │          │
│ 2. HTTP request ─┼─(Bearer token)──>│ 3. Validate │                │          │
│    with token    │  X-Auth-Type     │    token ───┼──TokenReview──>│          │
│                  │                  │             │<───Result──────│          │
│                  │                  │             │                │          │
│                  │                  │ 4. Extract  │                │          │
│                  │                  │    username │                │          │
│                  │                  │    (SA name)│                │          │
│                  │                  │             │                │          │
│                  │                  │ 5. Parse    │                │          │
│                  │                  │    developer│                │          │
│                  │                  │    from SA  │                │          │
│                  │<────Response─────│             │                │          │
└──────────────────┘                  └─────────────┘                └──────────┘
```

---

## Implementation Steps

### ✅ Step 0: Planning & Documentation

- [x] Create implementation plan document
- [x] Define architecture and API design
- [x] Establish file structure

### ✅ Step 1: Auth Framework

- [x] `internal/auth/identity.go` - Identity struct and context helpers
- [x] `internal/auth/provider.go` - AuthProvider interface
- [x] `internal/auth/k8s_sa_provider.go` - K8s SA token validation via TokenReview
- [x] `internal/auth/middleware.go` - HTTP auth middleware with zero-trust
- [x] `internal/k8s/auth.go` - ValidateToken, DeletePod, GetPodByName methods

**Completed:** November 19, 2025

### ⬜ Step 1: Auth Framework

**Files to create:**

- `internal/auth/identity.go` - Identity struct and types
- `internal/auth/provider.go` - AuthProvider interface
- `internal/auth/k8s_sa_provider.go` - K8s SA token validation
- `internal/auth/middleware.go` - HTTP auth middleware

**Key components:**

```go
// Identity representation
type Identity struct {
    Type       string            // "k8s-sa"
    Username   string            // Full SA name
    Developer  string            // Resolved developer name
    Namespace  string            // SA namespace
    Attributes map[string]string // Extra metadata
}

// Provider interface
type AuthProvider interface {
    Authenticate(ctx context.Context, token string) (*Identity, error)
    Name() string
    Type() string // Returns "k8s-sa", "github", etc.
}

// K8s SA Provider
type K8sSAProvider struct {
    client       *k8s.Client
    audience     string // "devenv-manager"
    namePattern  string // "devenv-{developer}"
}
```

**Deliverables:**

- Token validation via K8s TokenReview API
- SA name parsing (e.g., `devenv-eywalker` → `eywalker`)
- HTTP middleware that validates on every request (zero-trust)
- Support for X-Auth-Type header

---

### ⬜ Step 2: K8s Client Extensions

**Files to modify/create:**

- `internal/k8s/auth.go` - New file for auth-related K8s operations
- `internal/k8s/client.go` - Add pod deletion methods

**New methods:**

```go
// TokenReview validation
func (c *Client) ValidateToken(ctx context.Context, token, audience string) (*TokenReviewResult, error)

// Pod operations
func (c *Client) DeletePod(ctx context.Context, namespace, name string) error
func (c *Client) GetPodByName(ctx context.Context, namespace, name string) (*corev1.Pod, error)
```

**Deliverables:**

- TokenReview API integration
- Pod deletion functionality
- Pod retrieval by name/namespace

---

### ⬜ Step 3: API Server Implementation

**Files to create:**

- `internal/api/server.go` - HTTP server setup
- `internal/api/handlers.go` - Request handlers
- `internal/api/types.go` - Request/response structs
- `internal/api/errors.go` - Error handling

**API Endpoints:**

```
GET  /api/v1/pods              - List pods (filtered by authenticated developer)
GET  /api/v1/pods/{namespace}/{name} - Get specific pod
DELETE /api/v1/pods/{namespace}/{name} - Delete pod (if authorized)
GET  /api/v1/health            - Health check
GET  /api/v1/version           - Version info
```

**Request/Response format:**

```http
GET /api/v1/pods?namespace=default HTTP/1.1
Authorization: Bearer <k8s-sa-token>
X-Auth-Type: k8s-sa

Response:
{
  "pods": [
    {
      "name": "devenv-eywalker-0",
      "namespace": "default",
      "status": "Running",
      "restarts": 0,
      "age": "2d",
      "developer": "eywalker"
    }
  ]
}
```

**Deliverables:**

- chi router setup
- Middleware chain (logging → auth → handlers)
- JSON request/response handling
- Graceful shutdown

---

### ⬜ Step 4: Manager Server Command

**Files to modify/create:**

- `cmd/manager/server.go` - New server command
- `cmd/manager/root.go` - Register server subcommand

**Command usage:**

```bash
manager server --port 8080 --audience devenv-manager
```

**Flags:**

- `--port` (default: 8080)
- `--audience` (default: devenv-manager)
- `--config` (ConfigMap path for future providers)
- `--bind` (default: 0.0.0.0)

**Deliverables:**

- Server command with configuration
- Provider initialization
- Server lifecycle management

---

### ⬜ Step 5: Manager HTTP Client

**Files to create:**

- `internal/manager/client.go` - HTTP client for devenv CLI
- `internal/manager/types.go` - Shared API types

**Client functionality:**

```go
type Client struct {
    baseURL    string
    httpClient *http.Client
    tokenPath  string // Default: /var/run/secrets/tokens/devenv-manager
}

func (c *Client) ListPods(ctx context.Context) ([]Pod, error)
func (c *Client) DeletePod(ctx context.Context, name, namespace string) error
```

**Deliverables:**

- Token reading from projected volume
- HTTP client with proper headers (Authorization, X-Auth-Type)
- Error handling and retries
- JSON parsing

---

### ⬜ Step 6: DevENV Remote Commands

**Files to modify:**

- `cmd/devenv/pods.go` - Add remote functionality
- `cmd/devenv/root.go` - Add global flags

**New functionality:**

```bash
# Remote mode (via manager API)
devenv pods list --remote
devenv pods delete <name> --remote

# Direct mode (existing, via K8s API)
devenv pods list
devenv pods delete <name>
```

**Global flags:**

- `--remote` - Use manager API instead of direct K8s access
- `--manager-url` (default: http://manager-service:8080, or env DEVENV_MANAGER_URL)

**Deliverables:**

- Remote/local mode switching
- Manager client integration
- Consistent output format

---

### ⬜ Step 7: Deployment Manifests & Generation

**Files to create/modify:**

- `deploy/manager/serviceaccount.yaml` - Manager SA
- `deploy/manager/rbac.yaml` - RBAC for manager
- `deploy/manager/deployment.yaml` - Manager deployment
- `deploy/manager/service.yaml` - Manager service
- `cmd/devenv/generate.go` - Update generation logic
- `internal/templates/` - Add new templates

**Generation Logic Updates:**

1.  **Per-Developer ServiceAccount:**
    - `devenv generate <user>` should generate a `ServiceAccount` manifest for that user.
    - The SA name should be `devenv-<user>`.
    - It should be output alongside the pod manifest.

2.  **Manager Deployment:**
    - `devenv generate` (system manifests) should generate the Manager deployment manifests.
    - These should be output to the system output directory (e.g., `build/manager/`).

**RBAC Requirements:**

Manager ServiceAccount needs:

```yaml
# Create TokenReviews (validate incoming tokens)
- apiGroups: ["authentication.k8s.io"]
resources: ["tokenreviews"]
verbs: ["create"]

# List/Get/Delete pods
- apiGroups: [""]
resources: ["pods"]
verbs: ["list", "get", "delete"]
```

**DevENV Pod spec updates:**

```yaml
spec:
serviceAccountName: devenv-eywalker
volumes:
    - name: manager-token
    projected:
        sources:
        - serviceAccountToken:
            path: token
            audience: devenv-manager
            expirationSeconds: 3600
containers:
    - name: devenv
    volumeMounts:
        - name: manager-token
        mountPath: /var/run/secrets/tokens
        readOnly: true
```

**Deliverables:**

- Updated `devenv generate` command
- Templates for ServiceAccount and Manager deployment
- Complete K8s manifests generated by the tool

---

### ⬜ Step 8: Testing & Documentation

**Files to create:**

- `internal/auth/*_test.go` - Unit tests
- `internal/api/*_test.go` - API tests
- Integration test script
- Updated README sections

**Test coverage:**

- Token validation (valid, expired, wrong audience)
- Developer name extraction
- Authorization (correct developer, wrong developer)
- API endpoints
- Error cases

**Deliverables:**

- Unit tests (>80% coverage)
- Integration tests
- Documentation updates
- Usage examples

---

## File Structure (Final)

```
internal/
├── auth/
│   ├── identity.go
│   ├── provider.go
│   ├── k8s_sa_provider.go
│   ├── middleware.go
│   └── *_test.go
├── api/
│   ├── server.go
│   ├── handlers.go
│   ├── types.go
│   ├── errors.go
│   └── *_test.go
├── k8s/
│   ├── client.go
│   ├── auth.go
│   └── *_test.go
└── manager/
    ├── client.go
    ├── types.go
    └── *_test.go

cmd/
├── manager/
│   ├── main.go
│   ├── root.go
│   ├── version.go
│   ├── pods.go
│   └── server.go
└── devenv/
    ├── main.go
    ├── root.go
    ├── pods.go (updated)
    └── ...

deploy/
├── manager/
│   ├── serviceaccount.yaml
│   ├── rbac.yaml
│   ├── deployment.yaml
│   └── service.yaml
└── devenv/
    └── serviceaccount-template.yaml

docs/
└── manager-api-implementation.md (this file)
```

---

## Design Decisions

### Authentication

- **Zero-trust:** Token validated on every request, no sessions
- **Projected tokens:** Short-lived (1h), audience-bound, auto-rotating
- **Pluggable:** Provider interface allows future GitHub/OIDC support
- **X-Auth-Type header:** Client can specify auth type

### Authorization

- **Developer-scoped:** Users can only see/delete their own pods
- **Label-based:** Pods must have `developer=<name>` label
- **SA naming:** Pattern `devenv-{developer}` maps to developer name

### API Design

- **RESTful:** Standard HTTP verbs and status codes
- **JSON:** All requests/responses use JSON
- **Versioned:** `/api/v1/` prefix for future compatibility
- **Stateless:** No server-side session state

### Future Extensions (not in Phase 1)

- GitHub OAuth provider
- OIDC provider
- Developer mapping ConfigMap
- Additional endpoints (logs, exec, port-forward)
- Web UI

---

## Progress Tracking

**Current Step:** 0 (Planning Complete)  
**Next Step:** 1 (Auth Framework)  
**Estimated Completion:** TBD

---

## Notes & Decisions Log

- **2025-11-19:** Initial plan created
- **2025-11-19:** Decided on zero-trust (no sessions) approach
- **2025-11-19:** ConfigMap chosen for future mapping storage
- **2025-11-19:** X-Auth-Type header for provider selection
- **2025-11-19:** K8s SA provider as Phase 1 implementation
