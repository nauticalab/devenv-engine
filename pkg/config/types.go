package config

// SystemConfig represents the system-wide configuration
type SystemConfig struct {
	Organization   OrgConfig            `yaml:"organization"`
	Defaults       Defaults             `yaml:"defaults"`
	PortManagement PortManagementConfig `yaml:"portManagement"`
	Profiles       map[string]Profile   `yaml:"profiles"`
	Authentication AuthConfig           `yaml:"authentication"`
	CommonVolumes  []Volume             `yaml:"commonVolumes"`
	ResourceLimits map[string]Resources `yaml:"resourceLimits"`
	Cluster        ClusterConfig        `yaml:"cluster"`
}

// OrgConfig contains organization-specific settings.
type OrgConfig struct {
	Name         string `yaml:"name"`
	Domain       string `yaml:"domain"`
	SupportEmail string `yaml:"supportEmail"`
}

// Defaults contains system-wide default values.
type Defaults struct {
	Resources Resources         `yaml:"resources"`
	Packages  Packages          `yaml:"packages"`
	Volumes   []Volume          `yaml:"volumes"`
	Image     string            `yaml:"image"`
	HTTPPort  int               `yaml:"httpPort"`
	Env       map[string]string `yaml:"env"`
}

// PortManagementConfig defines port management settings.
type PortManagementConfig struct {
	SSHPortRange  PortRange      `yaml:"sshPortRange"`
	ReservedPorts []string       `yaml:"reservedPorts"`
	Assignments   map[string]int `yaml:"assignments"`
}

// PortRange defines a range of allowed ports
type PortRange struct {
	Start int `yaml:"start"`
	End   int `yaml:"end"`
}

// Profile defines an environment profile.
type Profile struct {
	Name         string            `yaml:"name"`
	Extends      string            `yaml:"extends,omitempty"`
	Resources    Resources         `yaml:"resources,omitempty"`
	Packages     Packages          `yaml:"packages,omitempty"`
	Volumes      []Volume          `yaml:"volumes,omitempty"`
	Env          map[string]string `yaml:"env,omitempty"`
	NodeSelector map[string]string `yaml:"nodeSelector,omitempty"`
}

// Resources defines compute resource requirements.
type Resources struct {
	CPU     string `yaml:"cpu"`
	Memory  string `yaml:"memory"`
	Storage string `yaml:"storage"`
	GPU     int    `yaml:"gpu"`
}

// Packages defines software packages to install
type Packages struct {
	Python []string `yaml:"python"`
	Apt    []string `yaml:"apt"`
	Npm    []string `yaml:"npm"`
}

// Volume defines a volume mount
type Volume struct {
	Name          string `yaml:"name"`
	LocalPath     string `yaml:"localPath"`
	ContainerPath string `yaml:"containerPath"`
	Description   string `yaml:"description"`
}

// AuthConfig defines authentication settings.
type AuthConfig struct {
	Type   string         `yaml:"type"`
	Config map[string]any `yaml:"config"`
}

// ClusterConfig defines Kubernetes cluster settings.
type ClusterConfig struct {
	Namespace       string          `yaml:"namespace"`
	ServiceAccount  string          `yaml:"serviceAccount"`
	PriorityClasses []PriorityClass `yaml:"priorityClasses"`
	IngressDomain   string          `yaml:"ingressDomain"`
	TLSSecret       string          `yaml:"tlsSecret"`
}

// PriorityClass defines a Kubernetes priority class.
type PriorityClass struct {
	Name        string `yaml:"name"`
	Value       int    `yaml:"value"`
	Description string `yaml:"description"`
}

// UserConfig represents a developer's main configuration.
type UserConfig struct {
	Name         string       `yaml:"name"`
	SSHPublicKey []string     `yaml:"sshPublicKey"`
	UID          int          `yaml:"uid"`
	IsAdmin      bool         `yaml:"isAdmin"`
	Defaults     UserDefaults `yaml:"defaults"`
	Environments []string     `yaml:"environments"`
	Dotfiles     []any        `yaml:"dotfiles"`
}

// UserDefaults contains user-specific default values.
type UserDefaults struct {
	Git       GitConfig `yaml:"git"`
	Resources Resources `yaml:"resources,omitempty"`
	Packages  Packages  `yaml:"packages,omitempty"`
	Volumes   []Volume  `yaml:"volumes,omitempty"`
}

// GitConfig contains Git user settings
type GitConfig struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

// EnvirionmentConfig represents a specdific environment configuration
type EnvironmentConfig struct {
	Name            string            `yaml:"name"`
	Profile         string            `yaml:"profile"`
	Resources       Resources         `yaml:"resources,omitempty"`
	Packages        Packages          `yaml:"packages,omitempty"`
	Volumes         []Volume          `yaml:"volumes,omitempty"`
	Ports           Ports             `yaml:"ports,omitempty"`
	Env             map[string]string `yaml:"env,omitempty"`
	NodeSelector    map[string]string `yaml:"nodeSelector,omitempty"`
	InitCommands    []string          `yaml:"initCommands,omitempty"`
	SecurityContext map[string]any    `yaml:"securityContext,omitempty"`
}

// CustomPort defines a custom network port
type Ports struct {
	SSH    int          `yaml:"ssh,omitempty"`
	HTTP   int          `yaml:"http,omitempty"`
	Custom []CustomPort `yaml:"custom,omitempty"`
}

type CustomPort struct {
	Name          string `yaml:"name"`
	ContainerPort int    `yaml:"containerPort"`
	ServicePort   int    `yaml:"servicePort"`
}

// CompleteEnvironment represents a fully merged environment configuration
type CompleteEnvironment struct {
	// Identification
	Name     string
	UserName string
	EnvName  string

	// Core configuration
	Resources    Resources
	Packages     Packages
	Volumes      []Volume
	Ports        Ports
	Env          map[string]string
	NodeSelector map[string]string
	InitCommands []string

	// User settings
	SSHPublicKey []string
	UID          int
	IsAdmin      bool
	GitConfig    GitConfig

	// Additional settings
	SecurityContext map[string]any
	Dotfiles        []any
}
