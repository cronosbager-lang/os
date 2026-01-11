package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	System  SystemConfig  `toml:"system"`
	IPC     IPCConfig     `toml:"ipc"`
	Services []ServiceConfig `toml:"services"`
}

type SystemConfig struct {
	Hostname    string `toml:"hostname"`
	LogLevel    string `toml:"log_level"`
	StorePath   string `toml:"store_path"`
}

type IPCConfig struct {
	SocketPath  string `toml:"socket_path"`
	Protocol    string `toml:"protocol"` // "protobuf" or "json"
	MaxConns    int    `toml:"max_connections"`
	Timeout     int    `toml:"timeout_ms"`
}

type ServiceConfig struct {
	Name      string   `toml:"name"`
	Command   string   `toml:"command"`
	Args      []string `toml:"args"`
	Type      string   `toml:"type"`
	Restart   string   `toml:"restart"`
	DependsOn []string `toml:"depends_on"`
	Env       map[string]string `toml:"env"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func Default() *Config {
	return &Config{
		System: SystemConfig{
			Hostname:  "mixos",
			LogLevel:  "info",
			StorePath: "/store",
		},
		IPC: IPCConfig{
			SocketPath: "/run/mixos/ipc.sock",
			Protocol:   "protobuf",
			MaxConns:   100,
			Timeout:    5000,
		},
		Services: []ServiceConfig{},
	}
}
