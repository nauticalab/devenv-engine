# DevEnv Engine

A Go-based tool for managing developer environments in Kubernetes.

## Overview

DevEnv Engine provides a command-line interface and library for:
- Managing developer environment configurations
- Rendering Kubernetes manifests
- Managing port assignments
- Validating configurations

## Getting Started

### Prerequisites

- Go 1.18+
- Kubernetes cluster
- kubectl
- git

### Installation

```bash
# Clone the repository
git clone https://github.com/walkerlab/devenv-engine.git
cd devenv-engine

# Build the CLI tool
make build

# Install the CLI tool
make install
```

### Basic Usage

```bash
# List all environments
devenv list

# Create a new environment
devenv create user-name environment-name

# Validate configurations
devenv validate

# Render Kubernetes manifests
devenv render --output ./output

## Documentation

For more details, see the [documentation](docs/README.md).

## License

MIT License
```