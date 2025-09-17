package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GlobalConfig represents organization-wide default configuration
// Only contains fields that make sense as global defaults
type GlobalConfig struct {
	Image        string         `yaml:"image,omitempty"`        // Default container image
	Resources    ResourceConfig `yaml:"resources,omitempty"`    // Default resource allocations
	Packages     PackageConfig  `yaml:"packages,omitempty"`     // Default packages to install
	Volumes      []VolumeMount  `yaml:"volumes,omitempty"`      // Default volume mounts
	SSHPublicKey any            `yaml:"sshPublicKey,omitempty"` // Can be string or []string
	UID          int            `yaml:"uid,omitempty"`          // Default user ID for container

	// Note: Fields like Name, SSHPublicKey, Git, UID, SSHPort are user-specific
	// and should NOT be in global config
}

// GetSSHKeys returns the SSH public keys as a normalized string slice.
// It handles both single string and string array formats from the YAML
// configuration, converting them to a consistent []string format.
//
// Returns an error if the SSH key field contains invalid data types
// or empty key values.
func (c *GlobalConfig) GetSSHKeys() ([]string, error) {
	return normalizeSSHKeys(c.SSHPublicKey)
}

// LoadGlobalConfig loads the global configuration file (devenv.yaml) from the config directory.
// If the file doesn't exist, returns an empty GlobalConfig without error.
func LoadGlobalConfig(configDir string) (*GlobalConfig, error) {
	globalConfigPath := filepath.Join(configDir, "devenv.yaml")
	return loadGlobalConfigFromPath(globalConfigPath)
}

// Add new function that accepts full path (for future flexibility)
func LoadGlobalConfigFromPath(globalConfigPath string) (*GlobalConfig, error) {
	return loadGlobalConfigFromPath(globalConfigPath)
}

// Extract common logic
func loadGlobalConfigFromPath(globalConfigPath string) (*GlobalConfig, error) {
	// Check if global config exists
	if _, err := os.Stat(globalConfigPath); os.IsNotExist(err) {
		// Return empty global config if file doesn't exist - this is not an error
		return &GlobalConfig{}, nil
	}

	// Read the global config file
	data, err := os.ReadFile(globalConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read global config file %s: %w", globalConfigPath, err)
	}

	// Parse the YAML
	var globalConfig GlobalConfig
	if err := yaml.Unmarshal(data, &globalConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in global config %s: %w", globalConfigPath, err)
	}

	return &globalConfig, nil
}

// MergeConfigs merges global defaults with user-specific configuration using field-dependent strategy:
//
// Override Fields (user replaces global):
//   - image, resources.cpu, resources.memory, resources.storage, resources.gpu
//
// Additive Fields (user extends global):
//   - packages.apt, packages.python, volumes, sshPublicKey, targetNodes
//
// User-Only Fields (preserved as-is):
//   - name, git, uid, sshPort, httpPort, isAdmin, skipAuth, refresh
func MergeConfigs(global *GlobalConfig, user *DevEnvConfig) *DevEnvConfig {
	// Start with user config as base (preserves all user-specific fields)
	merged := *user

	// === OVERRIDE FIELDS (user replaces global) ===

	// Image: use global default if user didn't specify
	if merged.Image == "" && global.Image != "" {
		merged.Image = global.Image
	}

	// Resources: merge each field individually, user overrides global
	mergeResources(&merged.Resources, &global.Resources)

	// === ADDITIVE FIELDS (user extends global) ===

	// Packages: combine global and user packages
	merged.Packages = mergePackages(&global.Packages, &user.Packages)

	// Volumes: combine global and user volumes
	merged.Volumes = mergeVolumes(global.Volumes, user.Volumes)

	globalSSHKeys, err := global.GetSSHKeys()
	if err != nil {
		// If global SSH keys are invalid, log error but proceed with user config
		fmt.Fprintf(os.Stderr, "warning: invalid global SSH keys: %v\n", err)
		globalSSHKeys = []string{}
	}

	if merged.UID == 0 && global.UID != 0 {
		merged.UID = global.UID
	}

	userSSHKeys := user.GetSSHKeysSlice()

	mergedSSHKeys := mergeStringSlices(globalSSHKeys, userSSHKeys)
	// SSH Keys and Target Nodes are already additive by nature in user config
	// (users can specify multiple keys/nodes)

	merged.SSHPublicKey = mergedSSHKeys

	// Optional: Validate that we have at least one SSH key after merging
	// Since DevEnvConfig should always have at least one SSH key (validated on load),
	// we log a warning if both global and user configs lack SSH keys.
	// This is not a hard error here, but may lead to issues later.
	if len(mergedSSHKeys) == 0 {
		fmt.Fprintf(os.Stderr, "warning: no SSH keys found for user %s (neither global nor user config has SSH keys)\n", merged.Name)
	}

	return &merged
}

// mergeResources handles resource field merging - user values override global defaults
func mergeResources(userResources *ResourceConfig, globalResources *ResourceConfig) {
	// CPU: user overrides global, or use global if user didn't specify
	if globalResources.CPU != nil {
		switch value := userResources.CPU.(type) {
		case int:
			if value == 0 {
				userResources.CPU = globalResources.CPU
			}
		case float64:
			if value <= 0.01 { // Treat near-zero as unspecified (floating point precision)
				userResources.CPU = globalResources.CPU
			}
		case string:
			if value == "" || value == "0" || value == "0.0" {
				userResources.CPU = globalResources.CPU
			}
		default:
			userResources.CPU = globalResources.CPU
		}
	}

	// Memory: user overrides global, or use global if user didn't specify
	if userResources.Memory == "" && globalResources.Memory != "" {
		userResources.Memory = globalResources.Memory
	}

	// Storage: user overrides global, or use global if user didn't specify
	if userResources.Storage == "" && globalResources.Storage != "" {
		userResources.Storage = globalResources.Storage
	}

	// GPU: user overrides global, or use global if user didn't specify
	// Note: GPU defaults to 0, so we need to check both configs
	if userResources.GPU == 0 && globalResources.GPU > 0 {
		userResources.GPU = globalResources.GPU
	}
}

// mergePackages combines global and user packages (additive strategy)
func mergePackages(globalPackages *PackageConfig, userPackages *PackageConfig) PackageConfig {
	merged := PackageConfig{}

	// APT packages: combine global + user, removing duplicates
	merged.APT = mergeStringSlices(globalPackages.APT, userPackages.APT)

	// Python packages: combine global + user, removing duplicates
	merged.Python = mergeStringSlices(globalPackages.Python, userPackages.Python)

	return merged
}

// mergeVolumes combines global and user volumes (additive strategy)
func mergeVolumes(globalVolumes []VolumeMount, userVolumes []VolumeMount) []VolumeMount {
	// Start with global volumes
	merged := make([]VolumeMount, len(globalVolumes))
	copy(merged, globalVolumes)

	// Add user volumes, avoiding duplicates by name
	globalVolumeNames := make(map[string]bool)
	for _, vol := range globalVolumes {
		globalVolumeNames[vol.Name] = true
	}

	for _, userVol := range userVolumes {
		if !globalVolumeNames[userVol.Name] {
			merged = append(merged, userVol)
		}
		// Note: If user specifies volume with same name as global,
		// global takes precedence. This prevents conflicts.
	}

	return merged
}

// mergeStringSlices combines two string slices, removing duplicates
func mergeStringSlices(slice1, slice2 []string) []string {
	// Use map to track unique items
	unique := make(map[string]bool)
	var result []string

	// Add items from first slice
	for _, item := range slice1 {
		if !unique[item] {
			unique[item] = true
			result = append(result, item)
		}
	}

	// Add items from second slice (avoiding duplicates)
	for _, item := range slice2 {
		if !unique[item] {
			unique[item] = true
			result = append(result, item)
		}
	}

	return result
}
