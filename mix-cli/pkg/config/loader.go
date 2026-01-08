package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the CLI configuration
type Config struct {
	Agent    AgentConfig    `yaml:"agent"`
	Package  PackageConfig  `yaml:"package"`
	Docker   DockerConfig   `yaml:"docker"`
	VM       VMConfig       `yaml:"vm"`
	Logging  LoggingConfig  `yaml:"logging"`
	UI       UIConfig       `yaml:"ui"`
}

// AgentConfig represents agent configuration
type AgentConfig struct {
	URL     string `yaml:"url"`
	Timeout int    `yaml:"timeout"`
	APIKey  string `yaml:"api_key,omitempty"`
}

// PackageConfig represents package manager configuration
type PackageConfig struct {
	Backend      string   `yaml:"backend"`
	Repositories []string `yaml:"repositories"`
	CacheDir     string   `yaml:"cache_dir"`
	AutoConfirm  bool     `yaml:"auto_confirm"`
}

// DockerConfig represents Docker configuration
type DockerConfig struct {
	Socket    string `yaml:"socket"`
	Registry  string `yaml:"registry"`
	BuildArgs map[string]string `yaml:"build_args"`
}

// VMConfig represents VM configuration
type VMConfig struct {
	DefaultCPUs    int    `yaml:"default_cpus"`
	DefaultMemory  int    `yaml:"default_memory"`
	DefaultDisk    int    `yaml:"default_disk"`
	VMDir          string `yaml:"vm_dir"`
	ISODir         string `yaml:"iso_dir"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	File   string `yaml:"file"`
	Format string `yaml:"format"`
}

// UIConfig represents UI configuration
type UIConfig struct {
	Colors    bool   `yaml:"colors"`
	Spinner   bool   `yaml:"spinner"`
	Progress  bool   `yaml:"progress"`
	Theme     string `yaml:"theme"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	
	return &Config{
		Agent: AgentConfig{
			URL:     "http://localhost:8765",
			Timeout: 300,
		},
		Package: PackageConfig{
			Backend:      "native",
			Repositories: []string{"https://repo.mixos.dev/core"},
			CacheDir:     filepath.Join(homeDir, ".cache/mix-pkg"),
			AutoConfirm:  false,
		},
		Docker: DockerConfig{
			Socket:   "/var/run/docker.sock",
			Registry: "docker.io",
		},
		VM: VMConfig{
			DefaultCPUs:   2,
			DefaultMemory: 2048,
			DefaultDisk:   20,
			VMDir:         filepath.Join(homeDir, ".mixos/vms"),
			ISODir:        filepath.Join(homeDir, ".mixos/isos"),
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		UI: UIConfig{
			Colors:   true,
			Spinner:  true,
			Progress: true,
			Theme:    "default",
		},
	}
}

// Load loads configuration from file
func Load() (*Config, error) {
	config := DefaultConfig()
	
	// Try multiple config locations
	configPaths := []string{
		"/etc/mixos/cli.yaml",
		filepath.Join(os.Getenv("HOME"), ".config/mixos/cli.yaml"),
		filepath.Join(os.Getenv("HOME"), ".mixos/cli.yaml"),
		"cli.yaml",
	}
	
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			if err := loadFromFile(path, config); err != nil {
				return nil, fmt.Errorf("failed to load config from %s: %w", path, err)
			}
			break
		}
	}
	
	// Override with environment variables
	loadFromEnv(config)
	
	return config, nil
}

// LoadFromFile loads configuration from a specific file
func LoadFromFile(path string) (*Config, error) {
	config := DefaultConfig()
	if err := loadFromFile(path, config); err != nil {
		return nil, err
	}
	return config, nil
}

func loadFromFile(path string, config *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	
	return yaml.Unmarshal(data, config)
}

func loadFromEnv(config *Config) {
	if url := os.Getenv("MIXOS_AGENT_URL"); url != "" {
		config.Agent.URL = url
	}
	if apiKey := os.Getenv("MIXOS_AGENT_API_KEY"); apiKey != "" {
		config.Agent.APIKey = apiKey
	}
	if backend := os.Getenv("MIXOS_PKG_BACKEND"); backend != "" {
		config.Package.Backend = backend
	}
	if socket := os.Getenv("DOCKER_HOST"); socket != "" {
		config.Docker.Socket = socket
	}
	if level := os.Getenv("MIXOS_LOG_LEVEL"); level != "" {
		config.Logging.Level = level
	}
}

// Save saves configuration to file
func (c *Config) Save(path string) error {
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	return os.WriteFile(path, data, 0644)
}

// Get gets a configuration value by key
func (c *Config) Get(key string) (interface{}, error) {
	switch key {
	case "agent.url":
		return c.Agent.URL, nil
	case "agent.timeout":
		return c.Agent.Timeout, nil
	case "package.backend":
		return c.Package.Backend, nil
	case "package.auto_confirm":
		return c.Package.AutoConfirm, nil
	case "docker.socket":
		return c.Docker.Socket, nil
	case "vm.default_cpus":
		return c.VM.DefaultCPUs, nil
	case "vm.default_memory":
		return c.VM.DefaultMemory, nil
	case "logging.level":
		return c.Logging.Level, nil
	case "ui.colors":
		return c.UI.Colors, nil
	default:
		return nil, fmt.Errorf("unknown config key: %s", key)
	}
}

// Set sets a configuration value by key
func (c *Config) Set(key string, value interface{}) error {
	switch key {
	case "agent.url":
		c.Agent.URL = fmt.Sprintf("%v", value)
	case "agent.timeout":
		if v, ok := value.(int); ok {
			c.Agent.Timeout = v
		}
	case "package.backend":
		c.Package.Backend = fmt.Sprintf("%v", value)
	case "package.auto_confirm":
		if v, ok := value.(bool); ok {
			c.Package.AutoConfirm = v
		}
	case "docker.socket":
		c.Docker.Socket = fmt.Sprintf("%v", value)
	case "vm.default_cpus":
		if v, ok := value.(int); ok {
			c.VM.DefaultCPUs = v
		}
	case "vm.default_memory":
		if v, ok := value.(int); ok {
			c.VM.DefaultMemory = v
		}
	case "logging.level":
		c.Logging.Level = fmt.Sprintf("%v", value)
	case "ui.colors":
		if v, ok := value.(bool); ok {
			c.UI.Colors = v
		}
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

// ListKeys returns all configuration keys
func (c *Config) ListKeys() []string {
	return []string{
		"agent.url",
		"agent.timeout",
		"agent.api_key",
		"package.backend",
		"package.repositories",
		"package.cache_dir",
		"package.auto_confirm",
		"docker.socket",
		"docker.registry",
		"vm.default_cpus",
		"vm.default_memory",
		"vm.default_disk",
		"vm.vm_dir",
		"vm.iso_dir",
		"logging.level",
		"logging.file",
		"logging.format",
		"ui.colors",
		"ui.spinner",
		"ui.progress",
		"ui.theme",
	}
}
