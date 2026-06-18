# Welcome to DevEnv Engine

A Go-based tool for managing developer environments in Kubernetes.

## Overview

DevEnv Engine provisions personal, persistent Linux development environments in Kubernetes — one per developer. Each environment is an always-on pod the developer SSHs into directly, with their own home directory, pre-installed tooling, git identity, and access to shared cluster storage. Environments are declared in YAML and rendered to Kubernetes manifests, so the full setup is reproducible and version-controlled.

DevEnv Engine provides a CLI for:

- Rendering per-developer Kubernetes manifests (StatefulSet, SSH/HTTP services, startup scripts) from configuration files
- Managing shared cluster-wide defaults alongside per-developer overrides

---

## Getting Started

### Prerequisites

- **Go 1.24+**
- **[Task](https://taskfile.dev)** — the build tool used by this project (`brew install go-task` or see [taskfile.dev](https://taskfile.dev/installation/))
- **kubectl** — to apply generated manifests to a cluster
- **A Kubernetes cluster** with NodePort access (for SSH and HTTP services)
- **git**

### Installation

```bash
# Clone the repository
git clone https://github.com/nauticalab/devenv-engine.git
cd devenv-engine

# Build the binary (output: ./bin/devenv)
task build

# Optionally copy to a directory in $PATH
cp bin/devenv /usr/local/bin/devenv
```

---

## Directory Structure

DevEnv Engine reads configuration from a **config directory** (default: `./developers`). The expected layout is:

```
developers/
├── devenv.yaml              # Required: shared global config
├── alice/
│   └── devenv-config.yaml   # Required: per-developer config
└── bob/
    └── devenv-config.yaml
```

- `devenv.yaml` is **required**. The tool will error if it is not present in the config directory.
- Each developer subdirectory **must** contain a `devenv-config.yaml`. Subdirectories without this file are ignored by `generate`.

---

## Configuration Files

### Configuration Hierarchy

Settings are resolved in this order, with each layer overriding the one before it:

```
System defaults  →  devenv.yaml (global)  →  devenv-config.yaml (per developer)
```

Some fields do not follow this override behavior and are instead merged additively across layers. These are identified in the [Field Glossary](#field-glossary).

---

### `devenv.yaml` — Global Config (Required)

Place this file at the root of your config directory. It sets cluster-wide defaults inherited by all developers. All fields within it are optional.

```yaml
# devenv.yaml
image: "ubuntu:22.04"
uid: 10000
namespace: "devenv"
hostName: "devenv.example.com"

enableAuth: true
authURL: "https://gate.example.com/oauth2/auth"
authSignIn: "https://gate.example.com/oauth2/start"

# SSH keys added to every developer's authorized_keys (in addition to their own)
sshPublicKey:
  - "ssh-ed25519 AAAA... admin@example.com"

resources:
  cpu: 4
  memory: "16Gi"

packages:
  apt:
    - vim
    - git

volumes:
  - name: mnt
    localPath: /mnt
    containerPath: /mnt
```

See the [Field Glossary](#field-glossary) at the end of this document for all available fields.

---

### `devenv-config.yaml` — Developer Config (Required)

Each developer must have this file in their subdirectory. It inherits all global config values and can override or extend them.

```yaml
# developers/alice/devenv-config.yaml
name: alice                  # Required. Must be a valid hostname (lowercase, alphanumeric, hyphens)
sshPublicKey: "ssh-ed25519 AAAA... alice@laptop"
sshPort: 30100               # Kubernetes NodePort for SSH (30000–32767)
httpPort: 8080               # Port for HTTP/web access (1024–65535)

git:
  name: "Alice Smith"
  email: "alice@example.com"

resources:
  cpu: 8
  memory: "32Gi"
  gpu: 1

packages:
  apt:
    - python3-pip
  python:
    - numpy
    - pandas

volumes:
  - name: data
    localPath: /mnt/data
    containerPath: /data
```

See the [Field Glossary](#field-glossary) at the end of this document for all available fields, including those inherited from `devenv.yaml`.

---

## Workflow

### 1. Set up the config directory

```bash
mkdir -p developers/alice
```

### 2. Create the global config

```bash
# developers/devenv.yaml
cat > developers/devenv.yaml <<EOF
namespace: "my-devenv"
image: "ubuntu:22.04"
resources:
  cpu: 4
  memory: "16Gi"
packages:
  apt:
    - vim
    - git
EOF
```

### 3. Create a developer config

```bash
# developers/alice/devenv-config.yaml
cat > developers/alice/devenv-config.yaml <<EOF
name: alice
sshPublicKey: "ssh-ed25519 AAAA... alice@laptop"
sshPort: 30100
httpPort: 8080
git:
  name: "Alice Smith"
  email: "alice@example.com"
EOF
```

### 4. Generate the manifests (preview first with --dry-run)

```bash
# Preview what would be generated without writing any files
devenv generate alice --dry-run

# Generate for a single developer (output: ./build/alice/)
devenv generate alice

# Generate for all developers in parallel (output: ./build/<name>/)
devenv generate --all-developers

# Specify a different output directory
devenv generate alice --output ./manifests

# Specify a different config directory
devenv generate alice --config-dir /path/to/configs
```

Generated manifests are written to `<output-dir>/<developer-name>/`. System manifests (e.g. namespace) are written to `<output-dir>/` directly and are generated once for all developers.

### 5. Apply to the cluster

```bash
# Apply system manifests first (namespace, etc.)
kubectl apply -f ./build/

# Apply a developer's manifests
kubectl apply -f ./build/alice/

# Apply all at once
kubectl apply -R -f ./build/
```

---

## CLI Reference

### `devenv generate`

```
Usage: devenv generate [developer-name] [flags]

Flags:
  -o, --output string       Output directory for generated manifests (default: ./build)
      --config-dir string   Directory containing developer configs (default: ./developers)
      --dry-run             Show what would be generated without writing files
      --all-developers      Generate manifests for all developers in the config directory
      --no-cleanup          Skip deletion of files from previous runs before generating
  -v, --verbose             Enable verbose output
```

Either a developer name or `--all-developers` must be provided (not both).

### `devenv version`

```
Usage: devenv version
```

Prints version, git commit, build time, and Go version.

---

## Building

```bash
task build           # Debug binary → ./bin/devenv
task build:release   # Optimized binary → ./bin/devenv
task build:all       # Cross-platform binaries (linux, darwin-amd64, darwin-arm64, windows)
task test            # Run all tests
task clean           # Remove build artifacts
```

---

## Field Glossary

Fields marked **additive** are merged across layers rather than overridden — the global value and developer value are combined.

### `devenv.yaml` fields

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `image` | string | No | `ubuntu:22.04` | Container image for the environment. |
| `uid` | int | No | `1000` | Linux UID for the developer user inside the container (1000–65535). |
| `namespace` | string | No | `devenv` | Kubernetes namespace for all DevEnv resources. |
| `environmentName` | string | No | `development` | Label applied to generated manifests. |
| `hostName` | string | No | — | Cluster ingress hostname. |
| `enableAuth` | bool | No | `false` | Enable OAuth2 proxy authentication for web access. |
| `authURL` | string | No | — | OAuth2 auth URL. |
| `authSignIn` | string | No | — | OAuth2 sign-in URL. |
| `installHomebrew` | bool | No | `true` | Install Linuxbrew in the container on first start. |
| `clearLocalPackages` | bool | No | `false` | Remove local package caches on start. |
| `clearVSCodeCache` | bool | No | `false` | Clear VS Code server cache on start. |
| `pythonBinPath` | string | No | `/opt/venv/bin` | Absolute path to the Python virtual environment bin directory. |
| `resources.cpu` | int, float, or string | No | `2` | CPU limit and request. Accepts cores as int/float (`4`, `1.5`) or millicores as string (`"500m"`). |
| `resources.memory` | int or string | No | `8Gi` | Memory limit and request. Bare integers are interpreted as Gi. Accepts `"16Gi"`, `"512Mi"`, `16`, etc. |
| `resources.storage` | string | No | `20Gi` | Persistent storage size for the home directory volume. |
| `resources.gpu` | int | No | `0` | Number of GPUs to request (0–8). |
| `sshPublicKey` | string or list | No | — | **Additive.** One or more OpenSSH public keys added to every developer's `authorized_keys`. At least one key must be present after merging with the developer config. |
| `packages.apt` | list | No | — | **Additive.** APT packages to install on start. |
| `packages.python` | list | No | — | **Additive.** Python packages to install via pip on start. |
| `packages.brew` | list | No | — | **Additive.** Homebrew packages to install on start. |
| `volumes` | list | No | — | **Additive.** Host path volume mounts. See volume fields below. |
| `gitRepos` | list | No | — | Git repositories to clone on startup. See git repo fields below. |

### `devenv-config.yaml` fields

All `devenv.yaml` fields above are also valid here and override the global value. The following fields are exclusive to developer configs.

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `name` | string | **Yes** | — | Used as the Kubernetes resource name and pod hostname. Must be 1–63 chars, hostname format (lowercase, alphanumeric, hyphens). |
| `sshPublicKey` | string or list | **Yes** | — | **Additive.** One or more OpenSSH public keys. Combined with global keys. Accepted formats: `ssh-ed25519`, `ssh-rsa`, `ecdsa-sha2-nistp256/384/521`, `sk-ecdsa-sha2-nistp256@openssh.com`. |
| `sshPort` | int | No | — | Kubernetes NodePort for SSH access (30000–32767). |
| `httpPort` | int | No | — | Port for HTTP/web access (1024–65535). |
| `isAdmin` | bool | No | `false` | Grants the pod a Kubernetes service account with elevated permissions. |
| `skipAuth` | bool | No | `false` | Bypass OAuth2 auth for this developer. Only effective when `enableAuth: true`. |
| `targetNodes` | list | No | — | Schedule the pod on specific cluster nodes (hostname format). |
| `git.name` | string | No | — | Git author name configured inside the environment. |
| `git.email` | string | No | — | Git author email configured inside the environment. |
| `refresh.enabled` | bool | No | `false` | Enable scheduled environment refresh. |
| `refresh.schedule` | string | No | — | Cron expression for refresh schedule. |
| `refresh.type` | string | No | — | Refresh type identifier. |
| `refresh.preserveHome` | bool | No | `false` | Preserve the home directory across refreshes. |

### Sub-fields for `volumes` and `gitRepos`

The following tables expand on the two list fields introduced in the `devenv.yaml` table above. They apply equally when those fields are used in `devenv-config.yaml`.

### Volume fields (`volumes` list entries)

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | Yes | Alphanumeric, 1–63 chars. A developer entry with the same name as a global entry overrides it. |
| `localPath` | string | Yes | Absolute host path to mount. |
| `containerPath` | string | Yes | Absolute path inside the container. |

### Git repo fields (`gitRepos` list entries)

| Field | Type | Required | Notes |
|---|---|---|---|
| `url` | string | Yes | Repository URL. |
| `branch` | string | No | Branch to check out. |
| `tag` | string | No | Tag to check out. |
| `commitHash` | string | No | Commit hash to check out. |
| `directory` | string | No | Local directory name to clone into. |

Only one of `branch`, `tag`, or `commitHash` may be specified per entry.
