# Welcome to DevENV Engine

A Go-based tool for managing developer environments in Kubernetes with centralized API management.

Authored by NauticaLab with passion.

## Overview

DevENV Engine provides:

- **CLI Tool**: Command-line interface for managing developer environment configurations
- **Manager API**: Centralized HTTP API server for remote pod management with zero-trust authentication
- **Manifest Generation**: Automated Kubernetes manifest rendering from YAML configurations
- **Configuration Validation**: Schema validation for developer environment configurations

## Architecture

The system consists of two main components:

1. **DevENV CLI** (`devenv`): Client tool for generating manifests and managing pods
2. **DevENV Manager** (`manager`): HTTP API server running in-cluster for pod management

### Authentication & Authorization

- **Zero-trust authentication**: Every API request validates Kubernetes service account tokens via TokenReview API
- **Developer-scoped authorization**: Users can only access pods with matching `developer` labels
- **Projected tokens**: Pods mount audience-specific tokens (`devenv-manager`) for secure API access

## Getting Started

### Prerequisites

- Go 1.24+
- Kubernetes cluster
- kubectl
- git

### Installation

```bash
# Clone the repository
git clone https://github.com/nauticalab/devenv-engine.git
cd devenv-engine

# Build the CLI tools
task build         # Builds both devenv and manager binaries

# Or build individually
task build:devenv
task build:manager
```

## Usage

### Generating Developer Environments

```bash
# Generate Kubernetes manifests for a specific user
devenv generate eywalker

# Generate with custom output directory
devenv generate eywalker --output ./output

# Generate for all users
devenv generate --output ./output
```

This creates:

- ServiceAccount (`devenv-<username>`)
- StatefulSet with projected token volume
- Service (SSH and HTTP access)
- ConfigMaps and Secrets
- Ingress rules

### Managing Pods

#### Direct Kubernetes Access

```bash
# List your pods (requires kubectl access)
devenv pods list

# Delete a pod
devenv pods delete <pod-name>
```

#### Remote Management via Manager API

```bash
# List pods through manager API (from inside a devenv pod)
devenv pods list --remote

# Delete pod remotely
devenv pods delete <pod-name> --remote
```

The `--remote` flag uses the Manager API instead of direct Kubernetes access, which is useful when running from within developer pods that don't have cluster-wide permissions.

### Deploying the Manager

The DevENV Manager runs as a deployment in your Kubernetes cluster:

```bash
# Deploy the manager
kubectl apply -f deploy/manager/
```

This creates:

- Namespace (`devenv`)
- ServiceAccount (`devenv-manager`) with RBAC permissions
- Deployment (2 replicas for HA)
- Service (ClusterIP on port 8080)

See [deploy/manager/README.md](deploy/manager/README.md) for detailed deployment instructions.

### Configuration

Developer configurations are stored in `developers/<username>/devenv-config.yaml`:

```yaml
# Base configuration
name: eywalker
image: ubuntu:22.04
resources:
  cpu: 4
  memory: 16Gi
  storage: 50Gi

# Git repositories to clone
gitRepos:
  - url: https://github.com/myorg/myrepo.git
    branch: main
    directory: ~/projects/myrepo

# Package installation
packages:
  python:
    - numpy
    - pandas
  apt:
    - vim
    - tmux

# Manager API URL (optional, defaults to http://devenv-manager:8080)
managerURL: http://devenv-manager:8080
```

### Manager API Reference

The Manager API provides the following endpoints:

**Public Endpoints:**

- `GET /api/v1/health` - Health check
- `GET /api/v1/version` - Version information

**Authenticated Endpoints:**

- `GET /api/v1/pods` - List pods for authenticated developer
  - Query params: `namespace` (default: devenv), `labels`
- `DELETE /api/v1/pods/{namespace}/{name}` - Delete pod (if owner)

Authentication requires a valid Kubernetes service account token with audience `devenv-manager`.

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/auth/... -v
go test ./internal/api/... -v

# Run with coverage
go test ./... -cover
```

### Project Structure

```
.
├── cmd/
│   ├── devenv/          # DevEnv CLI commands
│   └── manager/         # Manager API server
├── internal/
│   ├── api/             # HTTP API handlers and server
│   ├── auth/            # Authentication providers
│   ├── config/          # Configuration parsing
│   ├── k8s/             # Kubernetes client wrapper
│   ├── manager/         # Manager HTTP client
│   └── templates/       # Manifest templates
├── deploy/
│   ├── manager/         # Manager deployment manifests
│   └── examples/        # Example configurations
└── developers/          # User configurations
```

## Contributing

Contributions are welcome! Please ensure:

- All tests pass (`go test ./...`)
- Code follows Go conventions (`gofmt`, `golint`)
- New features include tests

## License

See LICENSE file for details.
