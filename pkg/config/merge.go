package config

import "fmt"

// MergeConfigurations merges system, user, and environment configs into a complete configuration
func MergeConfigurations(sysConfig *SystemConfig, userConfig *UserConfig, envConfig *EnvironmentConfig) (*CompleteEnvironment, error) {
	// Start with a new complete environment
	result := &CompleteEnvironment{
		Name:     fmt.Sprintf("%s-%s", userConfig.Name, envConfig.Name),
		UserName: userConfig.Name,
		EnvName:  envConfig.Name,

		// Set user settings
		SSHPublicKey: userConfig.SSHPublicKey,
		UID:          userConfig.UID,
		IsAdmin:      userConfig.IsAdmin,
		GitConfig:    userConfig.Defaults.Git,

		// Initialize maps
		Env:          make(map[string]string),
		NodeSelector: make(map[string]string),
	}

	// Start with system defaults
	result.Resources = sysConfig.Defaults.Resources
	result.Packages = sysConfig.Defaults.Packages
	result.Volumes = sysConfig.Defaults.Volumes

	// Copy system default environment variables
	for k, v := range sysConfig.Defaults.Env {
		result.Env[k] = v
	}

	// Apply profile if specified
	if envConfig.Profile != "" {
		profile, err := GetProfile(sysConfig, envConfig.Profile)
		if err != nil {
			return nil, fmt.Errorf("failed to get profile %s: %w", envConfig.Profile, err)
		}

		// Apply profile settings
		if profile.Resources.CPU != "" {
			result.Resources.CPU = profile.Resources.CPU
		}
		if profile.Resources.Memory != "" {
			result.Resources.Memory = profile.Resources.Memory
		}
		if profile.Resources.Storage != "" {
			result.Resources.Storage = profile.Resources.Storage
		}
		if profile.Resources.GPU != 0 {
			result.Resources.GPU = profile.Resources.GPU
		}

		// Merge packages
		result.Packages.Python = mergeStringSlices(result.Packages.Python, profile.Packages.Python)
		result.Packages.Apt = mergeStringSlices(result.Packages.Apt, profile.Packages.Apt)
		result.Packages.Npm = mergeStringSlices(result.Packages.Npm, profile.Packages.Npm)

		// Merge volumes
		result.Volumes = mergeVolumes(result.Volumes, profile.Volumes)

		// Merge environment variables
		for k, v := range profile.Env {
			result.Env[k] = v
		}

		// Merge node selectors
		for k, v := range profile.NodeSelector {
			result.NodeSelector[k] = v
		}

		// ================ Apply user defaults ================
		if userConfig.Defaults.Resources.CPU != "" {
			result.Resources.CPU = userConfig.Defaults.Resources.CPU
		}
		if userConfig.Defaults.Resources.Memory != "" {
			result.Resources.Memory = userConfig.Defaults.Resources.Memory
		}
		if userConfig.Defaults.Resources.Storage != "" {
			result.Resources.Storage = userConfig.Defaults.Resources.Storage
		}
		if userConfig.Defaults.Resources.GPU != 0 {
			result.Resources.GPU = userConfig.Defaults.Resources.GPU
		}

		// Merge user packages
		result.Packages.Python = mergeStringSlices(result.Packages.Python, userConfig.Defaults.Packages.Python)
		result.Packages.Apt = mergeStringSlices(result.Packages.Apt, userConfig.Defaults.Packages.Apt)
		result.Packages.Npm = mergeStringSlices(result.Packages.Npm, userConfig.Defaults.Packages.Npm)

		// Merge user volumes
		result.Volumes = mergeVolumes(result.Volumes, userConfig.Defaults.Volumes)

		// ================ Apply environment specific settings ================
		if envConfig.Resources.CPU != "" {
			result.Resources.CPU = envConfig.Resources.CPU
		}
		if envConfig.Resources.Memory != "" {
			result.Resources.Memory = envConfig.Resources.Memory
		}
		if envConfig.Resources.Storage != "" {
			result.Resources.Storage = envConfig.Resources.Storage
		}
		if envConfig.Resources.GPU != 0 {
			result.Resources.GPU = envConfig.Resources.GPU
		}

		// Merge environment packages
		result.Packages.Python = mergeStringSlices(result.Packages.Python, envConfig.Packages.Python)
		result.Packages.Apt = mergeStringSlices(result.Packages.Apt, envConfig.Packages.Apt)
		result.Packages.Npm = mergeStringSlices(result.Packages.Npm, envConfig.Packages.Npm)

		// Merge environment volumes
		result.Volumes = mergeVolumes(result.Volumes, envConfig.Volumes)

		// Set ports
		result.Ports = envConfig.Ports

		// Merge environment variables
		for k, v := range envConfig.Env {
			result.Env[k] = v
		}

		// Merge node selectors
		for k, v := range envConfig.NodeSelector {
			result.NodeSelector[k] = v
		}

		// Set init commands
		result.InitCommands = envConfig.InitCommands

		// Set security context
		result.SecurityContext = envConfig.SecurityContext

		// Set dotfiles
		result.Dotfiles = userConfig.Dotfiles

		// Apply resource limits for non-admin users
		if !userConfig.IsAdmin && sysConfig.ResourceLimits != nil {
			limits, ok := sysConfig.ResourceLimits["default"]
			if ok {
				// Check CPU
				// TODO: handle more complex cases such as CPU "2" vs "2000m"
				if limits.CPU != "" && limits.CPU != "unlimited" {
					if result.Resources.CPU != "unlimited" && result.Resources.CPU > limits.CPU {
						// Log a warning
						// TODO: implement logging
						fmt.Printf("Warning: CPU limit exceeded. Requested: %s, Limit: %s\n", result.Resources.CPU, limits.CPU)
						result.Resources.CPU = limits.CPU
					}
				}

				if limits.Memory != "" && limits.Memory != "unlimited" {
					if result.Resources.Memory != "unlimited" && result.Resources.Memory > limits.Memory {
						// Log a warning
						// TODO: implement logging
						fmt.Printf("Warning: Memory limit exceeded. Requested: %s, Limit: %s\n", result.Resources.Memory, limits.Memory)
						result.Resources.Memory = limits.Memory
					}
				}

				// Check GPU
				if limits.GPU > 0 {
					if result.Resources.GPU > limits.GPU {
						// Log a warning
						// TODO: implement logging
						fmt.Printf("Warning: GPU limit exceeded. Requested: %d, Limit: %d\n", result.Resources.GPU, limits.GPU)
						result.Resources.GPU = limits.GPU
					}
				}
			}
		}
	}

	return result, nil
}
