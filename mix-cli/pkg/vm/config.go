package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// VMTemplate represents a VM template
type VMTemplate struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	OS          string            `json:"os"`
	CPUs        int               `json:"cpus"`
	Memory      int               `json:"memory"`
	DiskSize    int               `json:"disk_size"`
	ISO         string            `json:"iso,omitempty"`
	CloudInit   *CloudInitConfig  `json:"cloud_init,omitempty"`
	Network     NetworkConfig     `json:"network"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NetworkConfig represents VM network configuration
type NetworkConfig struct {
	Type    string `json:"type"` // user, bridge, nat
	Bridge  string `json:"bridge,omitempty"`
	MAC     string `json:"mac,omitempty"`
	IP      string `json:"ip,omitempty"`
	Gateway string `json:"gateway,omitempty"`
	DNS     string `json:"dns,omitempty"`
}

// CloudInitConfig represents cloud-init configuration
type CloudInitConfig struct {
	UserData     string            `json:"user_data,omitempty"`
	MetaData     string            `json:"meta_data,omitempty"`
	NetworkData  string            `json:"network_data,omitempty"`
	Users        []CloudInitUser   `json:"users,omitempty"`
	Packages     []string          `json:"packages,omitempty"`
	RunCmd       []string          `json:"runcmd,omitempty"`
	WriteFiles   []CloudInitFile   `json:"write_files,omitempty"`
}

// CloudInitUser represents a cloud-init user
type CloudInitUser struct {
	Name              string   `json:"name"`
	Groups            []string `json:"groups,omitempty"`
	Shell             string   `json:"shell,omitempty"`
	Sudo              string   `json:"sudo,omitempty"`
	SSHAuthorizedKeys []string `json:"ssh_authorized_keys,omitempty"`
	Password          string   `json:"passwd,omitempty"`
	LockPasswd        bool     `json:"lock_passwd"`
}

// CloudInitFile represents a file to write via cloud-init
type CloudInitFile struct {
	Path        string `json:"path"`
	Content     string `json:"content"`
	Permissions string `json:"permissions,omitempty"`
	Owner       string `json:"owner,omitempty"`
}

// PredefinedTemplates contains common VM templates
var PredefinedTemplates = map[string]VMTemplate{
	"minimal": {
		Name:        "minimal",
		Description: "Minimal VM with 1 CPU and 512MB RAM",
		CPUs:        1,
		Memory:      512,
		DiskSize:    10,
		Network:     NetworkConfig{Type: "user"},
	},
	"standard": {
		Name:        "standard",
		Description: "Standard VM with 2 CPUs and 2GB RAM",
		CPUs:        2,
		Memory:      2048,
		DiskSize:    20,
		Network:     NetworkConfig{Type: "user"},
	},
	"developer": {
		Name:        "developer",
		Description: "Developer VM with 4 CPUs and 4GB RAM",
		CPUs:        4,
		Memory:      4096,
		DiskSize:    50,
		Network:     NetworkConfig{Type: "user"},
	},
	"server": {
		Name:        "server",
		Description: "Server VM with 4 CPUs and 8GB RAM",
		CPUs:        4,
		Memory:      8192,
		DiskSize:    100,
		Network:     NetworkConfig{Type: "bridge"},
	},
}

// TemplateManager manages VM templates
type TemplateManager struct {
	templateDir string
}

// NewTemplateManager creates a new template manager
func NewTemplateManager() *TemplateManager {
	templateDir := filepath.Join(os.Getenv("HOME"), ".mixos/vm-templates")
	os.MkdirAll(templateDir, 0755)

	return &TemplateManager{
		templateDir: templateDir,
	}
}

// GetTemplate gets a template by name
func (tm *TemplateManager) GetTemplate(name string) (*VMTemplate, error) {
	// Check predefined templates first
	if template, ok := PredefinedTemplates[name]; ok {
		return &template, nil
	}

	// Check custom templates
	templatePath := filepath.Join(tm.templateDir, name+".json")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("template not found: %s", name)
	}

	var template VMTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &template, nil
}

// SaveTemplate saves a custom template
func (tm *TemplateManager) SaveTemplate(template *VMTemplate) error {
	templatePath := filepath.Join(tm.templateDir, template.Name+".json")
	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(templatePath, data, 0644)
}

// ListTemplates lists all available templates
func (tm *TemplateManager) ListTemplates() ([]VMTemplate, error) {
	var templates []VMTemplate

	// Add predefined templates
	for _, template := range PredefinedTemplates {
		templates = append(templates, template)
	}

	// Add custom templates
	entries, err := os.ReadDir(tm.templateDir)
	if err == nil {
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == ".json" {
				name := entry.Name()[:len(entry.Name())-5]
				if template, err := tm.GetTemplate(name); err == nil {
					templates = append(templates, *template)
				}
			}
		}
	}

	return templates, nil
}

// DeleteTemplate deletes a custom template
func (tm *TemplateManager) DeleteTemplate(name string) error {
	// Cannot delete predefined templates
	if _, ok := PredefinedTemplates[name]; ok {
		return fmt.Errorf("cannot delete predefined template: %s", name)
	}

	templatePath := filepath.Join(tm.templateDir, name+".json")
	return os.Remove(templatePath)
}

// GenerateCloudInit generates cloud-init configuration
func GenerateCloudInit(config *CloudInitConfig) (string, string, error) {
	// Generate user-data
	userData := "#cloud-config\n"

	if len(config.Users) > 0 {
		userData += "users:\n"
		for _, user := range config.Users {
			userData += fmt.Sprintf("  - name: %s\n", user.Name)
			if len(user.Groups) > 0 {
				userData += fmt.Sprintf("    groups: %v\n", user.Groups)
			}
			if user.Shell != "" {
				userData += fmt.Sprintf("    shell: %s\n", user.Shell)
			}
			if user.Sudo != "" {
				userData += fmt.Sprintf("    sudo: %s\n", user.Sudo)
			}
			if len(user.SSHAuthorizedKeys) > 0 {
				userData += "    ssh_authorized_keys:\n"
				for _, key := range user.SSHAuthorizedKeys {
					userData += fmt.Sprintf("      - %s\n", key)
				}
			}
			userData += fmt.Sprintf("    lock_passwd: %v\n", user.LockPasswd)
		}
	}

	if len(config.Packages) > 0 {
		userData += "packages:\n"
		for _, pkg := range config.Packages {
			userData += fmt.Sprintf("  - %s\n", pkg)
		}
	}

	if len(config.WriteFiles) > 0 {
		userData += "write_files:\n"
		for _, file := range config.WriteFiles {
			userData += fmt.Sprintf("  - path: %s\n", file.Path)
			userData += fmt.Sprintf("    content: |\n")
			for _, line := range splitLines(file.Content) {
				userData += fmt.Sprintf("      %s\n", line)
			}
			if file.Permissions != "" {
				userData += fmt.Sprintf("    permissions: '%s'\n", file.Permissions)
			}
			if file.Owner != "" {
				userData += fmt.Sprintf("    owner: %s\n", file.Owner)
			}
		}
	}

	if len(config.RunCmd) > 0 {
		userData += "runcmd:\n"
		for _, cmd := range config.RunCmd {
			userData += fmt.Sprintf("  - %s\n", cmd)
		}
	}

	// Generate meta-data
	metaData := "instance-id: iid-local01\nlocal-hostname: mixos-vm\n"

	return userData, metaData, nil
}

func splitLines(s string) []string {
	var lines []string
	current := ""
	for _, c := range s {
		if c == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
