# DevENV Engine

A Go-based tool for managing developer environments in Kubernetes.

## Overview

DevEnv Engine provides a command-line interface and library for:

- Managing developer environment configurations
- Rendering Kubernetes manifests
- Validating configurations

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

# Build the CLI tool
make build

# Install the CLI tool
make install
```

### Basic Usage

```bash
# Generate DevENV manifest files from default `developers` directory, targeting user `eywalker`
devenv generate eywalker

# Validate configurations for a specific user
devenv validate eywalker

# Validate all users' configurations
devenv validate

# Generate Kubernetes manifests for a specific user
devenv generate eywalker --output ./output

# Generate Kubernetes manifests for all users
devenv generate --output ./output
```
