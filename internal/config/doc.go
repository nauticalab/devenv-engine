// Package config provides functionality for loading, parsing, and validating
// developer environment configurations. It supports flexible YAML configuration
// parsing with type conversion and validation for Kubernetes development environments.
//
// The package handles complex configuration scenarios including flexible data types,
// SSH key normalization, and resource specification parsing. It provides a robust
// foundation for generating Kubernetes manifests for developer environments.
//
// # Basic Usage
//
// The main entry point is [LoadDeveloperConfig] which loads a developer's
// configuration from a YAML file and returns a validated [DevEnvConfig] struct:
//
//	config, err := config.LoadDeveloperConfig("./developers", "alice")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Access configuration values with automatic defaults
//	cpu := config.CPU()        // Returns "2" if not specified
//	memory := config.Memory()  // Returns "8Gi" if not specified
//	userID := config.GetUserID() // Returns "1000" if UID not specified
//
// # Flexible Type Handling
//
// The configuration system handles multiple input formats for the same logical value:
//
//	// CPU can be specified as string, integer, or float in YAML:
//	cpu: "4"     // string
//	cpu: 4       // integer
//	cpu: 4.0     // float64
//
//	// All result in config.CPU() returning "4"
//
// # SSH Key Management
//
// SSH keys support both single and multiple key formats:
//
//	# Single SSH key
//	sshPublicKey: "ssh-rsa AAAAB3NzaC1yc2E..."
//
//	# Multiple SSH keys
//	sshPublicKey:
//	  - "ssh-rsa AAAAB3NzaC1yc2E..."
//	  - "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5..."
//
// Access normalized SSH keys:
//
//	sshKeys, err := config.GetSSHKeys()
//	if err != nil {
//	    // Handle SSH key parsing errors
//	}
//
//	// Template-friendly methods (no error handling needed in templates)
//	keysSlice := config.GetSSHKeysSlice()  // []string
//	keysString := config.GetSSHKeysString() // newline-separated string
//
// # Configuration Validation
//
// All loaded configurations are automatically validated for:
//   - Required fields (name, sshPublicKey)
//   - SSH key format and content
//   - Type compatibility and conversion
//
// Invalid configurations return descriptive errors during loading.
//
// # Template Integration
//
// DevEnvConfig provides template-friendly methods that handle errors internally
// and return safe defaults, making them suitable for use in Go templates:
//
//	{{ .CPU }}              // Always returns a string
//	{{ .Memory }}           // Returns memory with default fallback
//	{{ .GetSSHKeysString }} // Returns all SSH keys as single string
//	{{ .GetUserID }}        // Returns UID as string with default
//
// # Directory Structure
//
// The package expects developer configurations in the following structure:
//
//	developers/
//	├── alice/
//	│   └── devenv-config.yaml
//	├── bob/
//	│   └── devenv-config.yaml
//	└── charlie/
//	    └── devenv-config.yaml
//
// Each developer directory contains their devenv-config.yaml file with their
// specific configuration settings.
package config
