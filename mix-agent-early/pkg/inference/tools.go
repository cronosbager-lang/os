package inference

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Tool represents an executable tool for the AI agent
type Tool struct {
	Name        string
	Description string
	Execute     func(params map[string]interface{}) (string, error)
}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools map[string]*Tool
}

// NewToolRegistry creates a new tool registry with default tools
func NewToolRegistry() *ToolRegistry {
	r := &ToolRegistry{
		tools: make(map[string]*Tool),
	}
	r.registerDefaultTools()
	return r
}

// Register adds a tool to the registry
func (r *ToolRegistry) Register(tool *Tool) {
	r.tools[tool.Name] = tool
}

// Get returns a tool by name
func (r *ToolRegistry) Get(name string) *Tool {
	return r.tools[name]
}

// Execute runs a tool by name with given parameters
func (r *ToolRegistry) Execute(name string, params map[string]interface{}) (string, error) {
	tool := r.tools[name]
	if tool == nil {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return tool.Execute(params)
}

// ExecuteToolCalls executes a list of tool calls
func (r *ToolRegistry) ExecuteToolCalls(calls []ToolCall) []ToolResult {
	results := make([]ToolResult, len(calls))
	for i, call := range calls {
		output, err := r.Execute(call.Name, call.Params)
		results[i] = ToolResult{
			Name:    call.Name,
			Output:  output,
			Success: err == nil,
			Error:   errToString(err),
		}
	}
	return results
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Name    string `json:"name"`
	Output  string `json:"output"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func errToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// registerDefaultTools registers the default early boot tools
func (r *ToolRegistry) registerDefaultTools() {
	// Hardware detection
	r.Register(&Tool{
		Name:        "detect_hardware",
		Description: "Detect system hardware",
		Execute:     toolDetectHardware,
	})

	// Module loading
	r.Register(&Tool{
		Name:        "load_module",
		Description: "Load a kernel module",
		Execute:     toolLoadModule,
	})

	// Find root filesystem
	r.Register(&Tool{
		Name:        "find_root",
		Description: "Find root filesystem",
		Execute:     toolFindRoot,
	})

	// Mount filesystem
	r.Register(&Tool{
		Name:        "mount_filesystem",
		Description: "Mount a filesystem",
		Execute:     toolMountFilesystem,
	})

	// Emergency shell
	r.Register(&Tool{
		Name:        "emergency_shell",
		Description: "Drop to emergency shell",
		Execute:     toolEmergencyShell,
	})

	// Execute command
	r.Register(&Tool{
		Name:        "execute_command",
		Description: "Execute a shell command",
		Execute:     toolExecuteCommand,
	})

	// Read file
	r.Register(&Tool{
		Name:        "read_file",
		Description: "Read contents of a file",
		Execute:     toolReadFile,
	})

	// List directory
	r.Register(&Tool{
		Name:        "list_directory",
		Description: "List directory contents",
		Execute:     toolListDirectory,
	})
}

func toolDetectHardware(params map[string]interface{}) (string, error) {
	var result strings.Builder

	// CPU info
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "model name") {
				result.WriteString("CPU: ")
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					result.WriteString(strings.TrimSpace(parts[1]))
				}
				result.WriteString("\n")
				break
			}
		}
	}

	// Memory info
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "MemTotal:") {
				result.WriteString("Memory: ")
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					result.WriteString(parts[1])
					result.WriteString(" kB\n")
				}
				break
			}
		}
	}

	// Block devices
	result.WriteString("Block devices:\n")
	if entries, err := os.ReadDir("/sys/block"); err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") {
				continue
			}
			result.WriteString("  - /dev/")
			result.WriteString(name)

			// Get size
			sizePath := filepath.Join("/sys/block", name, "size")
			if data, err := os.ReadFile(sizePath); err == nil {
				sectors := strings.TrimSpace(string(data))
				result.WriteString(" (")
				result.WriteString(sectors)
				result.WriteString(" sectors)")
			}
			result.WriteString("\n")
		}
	}

	// Network interfaces
	result.WriteString("Network interfaces:\n")
	if entries, err := os.ReadDir("/sys/class/net"); err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if name == "lo" {
				continue
			}
			result.WriteString("  - ")
			result.WriteString(name)
			result.WriteString("\n")
		}
	}

	return result.String(), nil
}

func toolLoadModule(params map[string]interface{}) (string, error) {
	moduleName, ok := params["module"].(string)
	if !ok || moduleName == "" {
		return "", fmt.Errorf("module name required")
	}

	// Use modprobe to load module
	cmd := exec.Command("modprobe", moduleName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to load module %s: %v\n%s", moduleName, err, output)
	}

	return fmt.Sprintf("Module %s loaded successfully", moduleName), nil
}

func toolFindRoot(params map[string]interface{}) (string, error) {
	var candidates []string

	// Check by label
	labelPath := "/dev/disk/by-label/MIXOS_ROOT"
	if _, err := os.Stat(labelPath); err == nil {
		target, _ := os.Readlink(labelPath)
		candidates = append(candidates, fmt.Sprintf("Label MIXOS_ROOT -> %s", target))
	}

	// Check common devices
	commonDevices := []string{
		"/dev/sda2", "/dev/sda1",
		"/dev/vda2", "/dev/vda1",
		"/dev/nvme0n1p2", "/dev/nvme0n1p1",
	}

	for _, dev := range commonDevices {
		if _, err := os.Stat(dev); err == nil {
			candidates = append(candidates, dev)
		}
	}

	if len(candidates) == 0 {
		return "No root filesystem candidates found", nil
	}

	var result strings.Builder
	result.WriteString("Root filesystem candidates:\n")
	for _, c := range candidates {
		result.WriteString("  - ")
		result.WriteString(c)
		result.WriteString("\n")
	}

	return result.String(), nil
}

func toolMountFilesystem(params map[string]interface{}) (string, error) {
	device, _ := params["device"].(string)
	mountpoint, _ := params["mountpoint"].(string)
	fstype, _ := params["fstype"].(string)

	if device == "" {
		device = "/dev/sda2"
	}
	if mountpoint == "" {
		mountpoint = "/newroot"
	}
	if fstype == "" {
		fstype = "ext4"
	}

	// Create mountpoint
	if err := os.MkdirAll(mountpoint, 0755); err != nil {
		return "", fmt.Errorf("failed to create mountpoint: %v", err)
	}

	// Mount
	cmd := exec.Command("mount", "-t", fstype, device, mountpoint)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("mount failed: %v\n%s", err, output)
	}

	return fmt.Sprintf("Mounted %s on %s (type: %s)", device, mountpoint, fstype), nil
}

func toolEmergencyShell(params map[string]interface{}) (string, error) {
	// This would typically exec into a shell
	// For safety, we just return a message
	return "Emergency shell requested. Use 'exec /bin/sh' to drop to shell.", nil
}

func toolExecuteCommand(params map[string]interface{}) (string, error) {
	command, ok := params["command"].(string)
	if !ok || command == "" {
		return "", fmt.Errorf("command required")
	}

	// Safety check - block dangerous commands
	dangerous := []string{"rm -rf /", "dd if=/dev/zero", "mkfs", "> /dev/sd"}
	for _, d := range dangerous {
		if strings.Contains(command, d) {
			return "", fmt.Errorf("command blocked for safety")
		}
	}

	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command failed: %v", err)
	}

	return string(output), nil
}

func toolReadFile(params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path required")
	}

	// Safety check - limit to certain paths
	allowedPrefixes := []string{"/proc/", "/sys/", "/etc/", "/var/log/"}
	allowed := false
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(path, prefix) {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("path not allowed")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Limit output size
	if len(data) > 4096 {
		data = data[:4096]
	}

	return string(data), nil
}

func toolListDirectory(params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		path = "/"
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			result.WriteString("d ")
		} else {
			result.WriteString("- ")
		}
		result.WriteString(entry.Name())
		result.WriteString("\n")
	}

	return result.String(), nil
}
