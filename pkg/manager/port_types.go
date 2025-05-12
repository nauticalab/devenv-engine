package manager

import (
	"sync"

	"github.com/enigma-brain/devenv-engine/pkg/config"
)

// PortAssignments holds all port assignments
type PortAssignments struct {
	Assignments map[string]map[string]map[string]int `yaml:"assignments"`
	Available   map[string]NextAvailable             `yaml:"available"`
}

// NextAvailable tracks the next available port in a range
type NextAvailable struct {
	NextAvailable int `yaml:"nextAvailable"`
}

// PortManager manages port assignments for developer environments
type PortManager struct {
	FilePath    string
	Assignments PortAssignments
	SysConfig   *config.SystemConfig
	mutex       sync.Mutex
}

// PortConflict represents a port assignment conflict
type PortConflict struct {
	Port     int
	DevName  string
	EnvName  string
	PortType string
	PortName string
}
