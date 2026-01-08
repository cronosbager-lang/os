package boot

import (
	"os"
	"os/exec"
	"strings"
)

func LoadModule(name string) error {
	// Check if already loaded
	if isModuleLoaded(name) {
		return nil
	}

	cmd := exec.Command("modprobe", name)
	return cmd.Run()
}

func isModuleLoaded(name string) bool {
	data, err := os.ReadFile("/proc/modules")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), name)
}

func UnloadModule(name string) error {
	cmd := exec.Command("modprobe", "-r", name)
	return cmd.Run()
}
