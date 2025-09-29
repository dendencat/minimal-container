package spec

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gomini/internal/util"
)

// Config represents a subset of the OCI runtime configuration
type Config struct {
	OCIVersion string  `json:"ociVersion"`
	Process    Process `json:"process"`
	Root       Root    `json:"root"`
	Hostname   string  `json:"hostname"`
	Mounts     []Mount `json:"mounts"`
	Linux      Linux   `json:"linux"`
}

// Process defines the container process configuration
type Process struct {
	Terminal     bool         `json:"terminal"`
	User         User         `json:"user"`
	Args         []string     `json:"args"`
	Env          []string     `json:"env"`
	Cwd          string       `json:"cwd"`
	Capabilities Capabilities `json:"capabilities"`
	Rlimits      []Rlimit     `json:"rlimits"`
}

// User defines user information for the container process
type User struct {
	UID int `json:"uid"`
	GID int `json:"gid"`
}

// Capabilities defines the process capabilities
type Capabilities struct {
	Bounding    []string `json:"bounding"`
	Effective   []string `json:"effective"`
	Inheritable []string `json:"inheritable"`
	Permitted   []string `json:"permitted"`
}

// Rlimit defines resource limits
type Rlimit struct {
	Type string `json:"type"`
	Hard int    `json:"hard"`
	Soft int    `json:"soft"`
}

// Root defines the root filesystem for the container
type Root struct {
	Path     string `json:"path"`
	Readonly bool   `json:"readonly"`
}

// Mount defines a mount point in the container
type Mount struct {
	Destination string   `json:"destination"`
	Type        string   `json:"type"`
	Source      string   `json:"source"`
	Options     []string `json:"options"`
}

// Linux contains Linux-specific configuration
type Linux struct {
	Resources  Resources   `json:"resources"`
	Namespaces []Namespace `json:"namespaces"`
}

// Resources defines container resource limits
type Resources struct {
	Memory Memory `json:"memory"`
	CPU    CPU    `json:"cpu"`
	Pids   Pids   `json:"pids"`
}

// Memory defines memory resource limits
type Memory struct {
	Limit int64 `json:"limit"`
}

// CPU defines CPU resource limits
type CPU struct {
	Quota  int64 `json:"quota"`
	Period int64 `json:"period"`
}

// Pids defines process count limits
type Pids struct {
	Limit int `json:"limit"`
}

// Namespace defines a namespace for the container
type Namespace struct {
	Type string `json:"type"`
}

// LoadConfig loads and parses a config.json file from the specified bundle directory
func LoadConfig(bundleDir string) (*Config, error) {
	configPath := filepath.Join(bundleDir, "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, util.NewPathError("read config", configPath, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, util.NewPathError("parse config", configPath, err)
	}

	// Validate required fields
	if err := validateConfig(&config); err != nil {
		return nil, util.NewPathError("validate config", configPath, err)
	}

	return &config, nil
}

// validateConfig performs basic validation on the loaded configuration
func validateConfig(config *Config) error {
	if config.OCIVersion == "" {
		return util.NewSimpleError("validate config", "missing ociVersion field")
	}

	if len(config.Process.Args) == 0 {
		return util.NewSimpleError("validate config", "missing process args")
	}

	if config.Root.Path == "" {
		return util.NewSimpleError("validate config", "missing root path")
	}

	return nil
}

// GetRootfsPath returns the absolute path to the container's root filesystem
func (c *Config) GetRootfsPath(bundleDir string) string {
	if filepath.IsAbs(c.Root.Path) {
		return c.Root.Path
	}
	return filepath.Join(bundleDir, c.Root.Path)
}