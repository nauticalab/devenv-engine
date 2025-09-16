package config

import "fmt"

// DevEnvConfig represents the complete configuration for a developer environment
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

// GetDeveloperDir returns the path tot he developer's directory
func (c *DevEnvConfig) GetDeveloperDir() string {
	return c.developerDir
}

func (c *DevEnvConfig) GetUserID() string {
	if c.UID != 0 {
		return fmt.Sprintf("%d", c.UID)
	}
	return "1000"
}

// GetSSHKeysSlice returns SSH keys as a slice for template use
// This is needed because templates can't handle the error return from GetSSHKeys()
func (c *DevEnvConfig) GetSSHKeysSlice() []string {
	keys, err := c.GetSSHKeys()
	if err != nil {
		return []string{} // Return empty slice on error
	}
	return keys
}
