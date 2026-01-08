package config

import (
	"fmt"
	"strconv"
	"strings"
)

// ConfigValue represents a typed configuration value
type ConfigValue struct {
	Key         string
	Value       interface{}
	Type        ValueType
	Description string
	Default     interface{}
	Validator   func(interface{}) error
}

// ValueType represents the type of a configuration value
type ValueType int

const (
	TypeString ValueType = iota
	TypeInt
	TypeBool
	TypeStringSlice
	TypeMap
)

// String returns the string representation of the value type
func (t ValueType) String() string {
	switch t {
	case TypeString:
		return "string"
	case TypeInt:
		return "int"
	case TypeBool:
		return "bool"
	case TypeStringSlice:
		return "[]string"
	case TypeMap:
		return "map"
	default:
		return "unknown"
	}
}

// ConfigSchema defines the configuration schema
var ConfigSchema = map[string]ConfigValue{
	"agent.url": {
		Key:         "agent.url",
		Type:        TypeString,
		Description: "URL of the MIXOS AI Agent API",
		Default:     "http://localhost:8765",
	},
	"agent.timeout": {
		Key:         "agent.timeout",
		Type:        TypeInt,
		Description: "Request timeout in seconds",
		Default:     300,
		Validator: func(v interface{}) error {
			if i, ok := v.(int); ok && i < 0 {
				return fmt.Errorf("timeout must be positive")
			}
			return nil
		},
	},
	"agent.api_key": {
		Key:         "agent.api_key",
		Type:        TypeString,
		Description: "API key for agent authentication",
		Default:     "",
	},
	"package.backend": {
		Key:         "package.backend",
		Type:        TypeString,
		Description: "Package manager backend (native, alpm, dpkg)",
		Default:     "native",
		Validator: func(v interface{}) error {
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("must be a string")
			}
			valid := []string{"native", "alpm", "dpkg"}
			for _, b := range valid {
				if s == b {
					return nil
				}
			}
			return fmt.Errorf("must be one of: %s", strings.Join(valid, ", "))
		},
	},
	"package.repositories": {
		Key:         "package.repositories",
		Type:        TypeStringSlice,
		Description: "List of package repository URLs",
		Default:     []string{"https://repo.mixos.dev/core"},
	},
	"package.cache_dir": {
		Key:         "package.cache_dir",
		Type:        TypeString,
		Description: "Directory for package cache",
		Default:     "~/.cache/mix-pkg",
	},
	"package.auto_confirm": {
		Key:         "package.auto_confirm",
		Type:        TypeBool,
		Description: "Automatically confirm package operations",
		Default:     false,
	},
	"docker.socket": {
		Key:         "docker.socket",
		Type:        TypeString,
		Description: "Docker socket path",
		Default:     "/var/run/docker.sock",
	},
	"docker.registry": {
		Key:         "docker.registry",
		Type:        TypeString,
		Description: "Default Docker registry",
		Default:     "docker.io",
	},
	"vm.default_cpus": {
		Key:         "vm.default_cpus",
		Type:        TypeInt,
		Description: "Default number of CPUs for new VMs",
		Default:     2,
		Validator: func(v interface{}) error {
			if i, ok := v.(int); ok && (i < 1 || i > 128) {
				return fmt.Errorf("CPUs must be between 1 and 128")
			}
			return nil
		},
	},
	"vm.default_memory": {
		Key:         "vm.default_memory",
		Type:        TypeInt,
		Description: "Default memory (MB) for new VMs",
		Default:     2048,
		Validator: func(v interface{}) error {
			if i, ok := v.(int); ok && (i < 256 || i > 1048576) {
				return fmt.Errorf("memory must be between 256 and 1048576 MB")
			}
			return nil
		},
	},
	"vm.default_disk": {
		Key:         "vm.default_disk",
		Type:        TypeInt,
		Description: "Default disk size (GB) for new VMs",
		Default:     20,
	},
	"logging.level": {
		Key:         "logging.level",
		Type:        TypeString,
		Description: "Logging level (debug, info, warn, error)",
		Default:     "info",
		Validator: func(v interface{}) error {
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("must be a string")
			}
			valid := []string{"debug", "info", "warn", "error"}
			for _, l := range valid {
				if s == l {
					return nil
				}
			}
			return fmt.Errorf("must be one of: %s", strings.Join(valid, ", "))
		},
	},
	"logging.file": {
		Key:         "logging.file",
		Type:        TypeString,
		Description: "Log file path",
		Default:     "",
	},
	"ui.colors": {
		Key:         "ui.colors",
		Type:        TypeBool,
		Description: "Enable colored output",
		Default:     true,
	},
	"ui.spinner": {
		Key:         "ui.spinner",
		Type:        TypeBool,
		Description: "Show spinner for long operations",
		Default:     true,
	},
	"ui.progress": {
		Key:         "ui.progress",
		Type:        TypeBool,
		Description: "Show progress bars",
		Default:     true,
	},
	"ui.theme": {
		Key:         "ui.theme",
		Type:        TypeString,
		Description: "UI theme (default, dark, light)",
		Default:     "default",
	},
}

// ParseValue parses a string value to the appropriate type
func ParseValue(key string, value string) (interface{}, error) {
	schema, ok := ConfigSchema[key]
	if !ok {
		return value, nil // Unknown keys are treated as strings
	}

	switch schema.Type {
	case TypeString:
		return value, nil
	case TypeInt:
		return strconv.Atoi(value)
	case TypeBool:
		return strconv.ParseBool(value)
	case TypeStringSlice:
		return strings.Split(value, ","), nil
	default:
		return value, nil
	}
}

// ValidateValue validates a configuration value
func ValidateValue(key string, value interface{}) error {
	schema, ok := ConfigSchema[key]
	if !ok {
		return nil // Unknown keys are not validated
	}

	if schema.Validator != nil {
		return schema.Validator(value)
	}

	return nil
}

// GetDescription returns the description for a config key
func GetDescription(key string) string {
	if schema, ok := ConfigSchema[key]; ok {
		return schema.Description
	}
	return ""
}

// GetDefault returns the default value for a config key
func GetDefault(key string) interface{} {
	if schema, ok := ConfigSchema[key]; ok {
		return schema.Default
	}
	return nil
}

// GetType returns the type for a config key
func GetType(key string) ValueType {
	if schema, ok := ConfigSchema[key]; ok {
		return schema.Type
	}
	return TypeString
}
