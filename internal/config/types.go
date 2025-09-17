package config

import "fmt"

// DevEnvConfig represents the complete configuration for a developer environment.
// It contains all the settings needed to generate Kubernetes manifests including
// resource allocation, SSH access, package installation, and volume mounts.
//
// The struct supports flexible field types where appropriate (e.g., CPU can be
// specified as string, int, or float64) and provides methods for accessing
// values with sensible defaults.
type DevEnvConfig struct {
	Name         string         `yamle:"name"`
	SSHPublicKey any            `yaml:"sshPublicKey"` // Can be string or []string
	SSHPort      int            `yaml:"sshPort,omitempty"`
	HTTPPort     int            `yaml:"httpPort,omitempty"`
	UID          int            `yaml:"uid,omitempty"`
	IsAdmin      bool           `yaml:"isAdmin,omitempty"`
	SkipAuth     bool           `yaml:"skipAuth,omitempty"`
	Image        string         `yaml:"image,omitempty"`
	TargetNodes  []string       `yaml:"targetNodes,omitempty"`
	Git          GitConfig      `yaml:"git,omitempty"`
	Packages     PackageConfig  `yaml:"packages,omitempty"`
	Resources    ResourceConfig `yaml:"resources,omitempty"`
	Volumes      []VolumeMount  `yaml:"volumes,omitempty"`
	Refresh      RefreshConfig  `yaml:"refresh,omitempty"`
	developerDir string         `yaml:"-"` // Directory where the developer config is located
}

// SystemDefaults holds default resource allocation values for developer environments.
// These values are used as fallbacks when resources are not specified in individual
// developer configurations. In future versions, these defaults will be configurable
// through global configuration files.
var SystemDefaults = struct {
	CPU    int
	Memory string
	UID    int
	Image  string
}{
	CPU:    2,
	Memory: "8Gi",
	UID:    1000,
	Image:  "ubuntu:22.04", // TODO: consider more sensible default
}

// GitConfig represents Git-related configuration
type GitConfig struct {
	Name  string `yaml:"name,omitempty"`
	Email string `yaml:"email,omitempty"`
}

// PackageConfig represents package installation configuration
type PackageConfig struct {
	Python []string `yaml:"python,omitempty"`
	APT    []string `yaml:"apt,omitempty"`
	// Consider adding other package managers such as NPM, Yarn, etc.
}

// ResourceConfig represetns resource allocation
type ResourceConfig struct {
	CPU     any    `yaml:"cpu,omitempty"` // Can be string or int
	Memory  string `yaml:"memory,omitempty"`
	Storage string `yaml:"storage,omitempty"`
	GPU     int    `yaml:"gpu,omitempty"` // Number of GPUs requested
}

// VolumeMount represents a volume mount configuration
type VolumeMount struct {
	Name          string `yaml:"name"`
	LocalPath     string `yaml:"localPath"`
	ContainerPath string `yaml:"containerPath"`
}

// RefreshConfig represents auto-refresh settings
type RefreshConfig struct {
	Enabled      bool   `yaml:"enabled,omitempty"`
	Schedule     string `yaml:"schedule,omitempty"` // Cron format
	Type         string `yaml:"type,omitempty"`
	PreserveHome bool   `yaml:"preserveHome,omitempty"`
}

// GetDeveloperDir returns the filesystem path to the developer's configuration directory.
// This path is set during configuration loading and points to the directory containing
// the developer's devenv-config.yaml file and any associated resources.
func (c *DevEnvConfig) GetDeveloperDir() string {
	return c.developerDir
}

// GetUserID returns the user ID as a string for use in Kubernetes manifests.
// Uses the configured UID value or returns DefaultValue.UID as the default
// if no UID is specified in the developer's configuration.
func (c *DevEnvConfig) GetUserID() string {
	if c.UID != 0 {
		return fmt.Sprintf("%d", c.UID)
	}
	return "1000"
}

// GPU returns the number of GPU resources requested for the developer environment.
// Returns 0 if no GPU allocation is specified in the configuration.
func (c *DevEnvConfig) GPU() int {
	return c.Resources.GPU
}

// CPU returns the CPU resource allocation as a string suitable for Kubernetes manifests.
// It handles flexible input types from YAML (string, int, float64) and converts them
// to a consistent string format.
//
// Returns the default value from DefaultValue.CPU if no CPU allocation is specified
// in the developer's configuration.
func (c *DevEnvConfig) CPU() string {
	// CPU can be string or number
	defaultCPU := fmt.Sprintf("%d", SystemDefaults.CPU)
	if c.Resources.CPU == nil {
		return defaultCPU
	}
	switch v := c.Resources.CPU.(type) {
	case string:
		if v == "" {
			return defaultCPU
		}
		return v
	case int:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		return defaultCPU
	}
}

// Memory returns the memory resource allocation as a string suitable for Kubernetes manifests.
// Uses the configured memory value or returns the default from DefaultValue.Memory
// if no memory allocation is specified in the developer's configuration.
func (c *DevEnvConfig) Memory() string {
	if c.Resources.Memory == "" {
		return SystemDefaults.Memory
	}
	return c.Resources.Memory
}

// CPURequest returns the CPU resource request as a string suitable for Kubernetes manifests.
// This is currently an alias for the CPU method, but separated for potential future
// differentiation between limits and requests.
func (c *DevEnvConfig) CPURequest() string {
	return c.CPU()
}

// MemoryRequest returns the memory resource request as a string suitable for Kubernetes manifests.
// This is currently an alias for the Memory method, but separated for potential future
// differentiation between limits and requests.
func (c *DevEnvConfig) MemoryRequest() string {
	return c.Memory()
}

// NodePort returns the SSH port number for NodePort service configuration.
// This is an alias for the SSHPort field, providing template-friendly access
// to the port value for Kubernetes NodePort services.
func (c *DevEnvConfig) NodePort() int {
	return c.SSHPort
}

// VolumeMounts returns the configured volume mount specifications.
// Returns the slice of VolumeMount configurations for binding local directories
// into the developer environment container.
func (c *DevEnvConfig) VolumeMounts() []VolumeMount {
	return c.Volumes
}

// GetSSHKeysSlice returns SSH keys as a string slice for use in Go templates.
// This method handles errors internally and returns an empty slice if SSH key
// parsing fails, making it safe for use in templates where error handling
// is not possible.
func (c *DevEnvConfig) GetSSHKeysSlice() []string {
	keys, err := c.GetSSHKeys()
	if err != nil {
		return []string{} // Return empty slice on error
	}
	return keys
}

// GetSSHKeysString returns all SSH keys as a single newline-separated string
// for use in Go templates. This format is suitable for writing to authorized_keys
// files or similar multi-line SSH key configurations.
func (c *DevEnvConfig) GetSSHKeysString() string {
	keys := c.GetSSHKeysSlice()
	result := ""
	for i, key := range keys {
		if i > 0 {
			result += "\n"
		}
		result += key
	}
	return result
}

// GetSSHKeys returns the SSH public keys as a normalized string slice.
// It handles both single string and string array formats from the YAML
// configuration, converting them to a consistent []string format.
//
// Returns an error if the SSH key field contains invalid data types
// or empty key values.
func (c *DevEnvConfig) GetSSHKeys() ([]string, error) {
	return normalizeSSHKeys(c.SSHPublicKey)
}
