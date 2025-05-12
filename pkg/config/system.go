package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadSystemConfig loads the system configuration from a file
func LoadSystemConfig(path string) (*SystemConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read system config file: %w", err)
	}

	var config SystemConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse system config: %w", err)
	}

	// Set defaults if not specified
	if config.Defaults.Image == "" {
		config.Defaults.Image = "ghcr.io/enigma-brain/python-scientific-gpu:latest"
	}

	if config.Defaults.HTTPPort == 0 {
		config.Defaults.HTTPPort = 8888
	}

	if config.PortManagement.SSHPortRange.Start == 0 {
		config.PortManagement.SSHPortRange.Start = 30000
	}

	if config.PortManagement.SSHPortRange.End == 0 {
		config.PortManagement.SSHPortRange.End = 32767
	}

	if config.Cluster.Namespace == "" {
		config.Cluster.Namespace = "enigma-devenvs"
	}

	return &config, nil
}

func GetProfile(sysConfig *SystemConfig, profileName string) (Profile, error) {
	profile, ok := sysConfig.Profiles[profileName]
	if !ok {
		return Profile{}, fmt.Errorf("profile %s not found", profileName)
	}

	// Resolve profile inheritance
	if profile.Extends != "" {
		parentProfile, err := GetProfile(sysConfig, profile.Extends)
		if err != nil {
			return Profile{}, fmt.Errorf("failed to resolve parent profile %s: %w", profile.Extends, err)
		}

		mergedProfile := parentProfile // Create a new profile to avoid modifying the original

		// Override with child profile settings
		if profile.Resources.CPU != "" {
			mergedProfile.Resources.CPU = profile.Resources.CPU
		}
		if profile.Resources.Memory != "" {
			mergedProfile.Resources.Memory = profile.Resources.Memory
		}
		if profile.Resources.Storage != "" {
			mergedProfile.Resources.Storage = profile.Resources.Storage
		}
		if profile.Resources.GPU != 0 {
			mergedProfile.Resources.GPU = profile.Resources.GPU
		}

		// Merge packages
		mergedProfile.Packages.Python = mergeStringSlices(parentProfile.Packages.Python, profile.Packages.Python)
		mergedProfile.Packages.Apt = mergeStringSlices(parentProfile.Packages.Apt, profile.Packages.Apt)
		mergedProfile.Packages.Npm = mergeStringSlices(parentProfile.Packages.Npm, profile.Packages.Npm)

		// Merge volumes
		mergedProfile.Volumes = mergeVolumes(parentProfile.Volumes, profile.Volumes)

		// Merge environment variables
		if mergedProfile.Env == nil {
			mergedProfile.Env = make(map[string]string)
		}
		for k, v := range profile.Env {
			mergedProfile.Env[k] = v
		}

		// Merge node selectors
		if mergedProfile.NodeSelector == nil {
			mergedProfile.NodeSelector = make(map[string]string)
		}
		for k, v := range profile.NodeSelector {
			mergedProfile.NodeSelector[k] = v
		}

		return mergedProfile, nil
	}

	return profile, nil
}

// given a list of string values, merge them into a single list
// ensuring that duplicates are removed
// TODO: consider special handling to detect string with shared parts such
// as "numpy=1.21.0" and "numpy=1.22.0" as duplicates, using the override
// version
func mergeStringSlices(base, override []string) []string {
	// Create a map for quick lookup
	seen := make(map[string]bool)
	for _, item := range base {
		seen[item] = true
	}

	// Add items from override that aren't in base
	for _, item := range override {
		if !seen[item] {
			base = append(base, item)
			seen[item] = true
		}
	}

	return base
}

// Helper function to merge volumes - ensuring all volumes are unique in
// the Name field
func mergeVolumes(base, override []Volume) []Volume {
	// Create a map for quick lookup
	volumeMap := make(map[string]Volume)
	for _, vol := range base {
		volumeMap[vol.Name] = vol
	}

	// Override or add volumes
	for _, vol := range override {
		volumeMap[vol.Name] = vol
	}

	// Convert the map back to a slice
	result := make([]Volume, 0, len(volumeMap))
	for _, vol := range volumeMap {
		result = append(result, vol)
	}

	return result
}
