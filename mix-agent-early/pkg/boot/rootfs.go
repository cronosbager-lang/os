package boot

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func FindRootFS() (string, error) {
	// Check kernel command line for root= parameter
	cmdline, err := os.ReadFile("/proc/cmdline")
	if err == nil {
		root := parseRootParam(string(cmdline))
		if root != "" {
			return resolveRootDevice(root)
		}
	}

	// Search for root filesystem by label
	device, err := findByLabel("MIXOS_ROOT")
	if err == nil {
		return device, nil
	}

	// Search for root filesystem by UUID
	device, err = findByUUID("")
	if err == nil {
		return device, nil
	}

	// Fallback: try common devices
	fallbacks := []string{
		"/dev/sda2",
		"/dev/vda2",
		"/dev/nvme0n1p2",
	}

	for _, dev := range fallbacks {
		if _, err := os.Stat(dev); err == nil {
			return dev, nil
		}
	}

	return "", fmt.Errorf("root filesystem not found")
}

func parseRootParam(cmdline string) string {
	for _, param := range strings.Fields(cmdline) {
		if strings.HasPrefix(param, "root=") {
			return strings.TrimPrefix(param, "root=")
		}
	}
	return ""
}

func resolveRootDevice(root string) (string, error) {
	// Handle different root= formats
	if strings.HasPrefix(root, "/dev/") {
		return root, nil
	}

	if strings.HasPrefix(root, "LABEL=") {
		label := strings.TrimPrefix(root, "LABEL=")
		return findByLabel(label)
	}

	if strings.HasPrefix(root, "UUID=") {
		uuid := strings.TrimPrefix(root, "UUID=")
		return findByUUID(uuid)
	}

	return root, nil
}

func findByLabel(label string) (string, error) {
	path := filepath.Join("/dev/disk/by-label", label)
	target, err := os.Readlink(path)
	if err != nil {
		return "", err
	}

	// Resolve relative path
	if !strings.HasPrefix(target, "/") {
		target = filepath.Join("/dev/disk/by-label", target)
	}

	return filepath.Clean(target), nil
}

func findByUUID(uuid string) (string, error) {
	if uuid == "" {
		return "", fmt.Errorf("no UUID specified")
	}

	path := filepath.Join("/dev/disk/by-uuid", uuid)
	target, err := os.Readlink(path)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(target, "/") {
		target = filepath.Join("/dev/disk/by-uuid", target)
	}

	return filepath.Clean(target), nil
}

func MountRoot(device, mountpoint string) error {
	// Create mount point if needed
	if err := os.MkdirAll(mountpoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	// Try different filesystem types
	fsTypes := []string{"ext4", "btrfs", "xfs"}

	for _, fsType := range fsTypes {
		cmd := exec.Command("mount", "-t", fsType, device, mountpoint)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	// Try without specifying type
	cmd := exec.Command("mount", device, mountpoint)
	return cmd.Run()
}

func ValidateRoot(mountpoint string) error {
	// Check for essential files/directories
	required := []string{
		"sbin/init",
		"bin",
		"lib",
		"etc",
	}

	for _, path := range required {
		fullPath := filepath.Join(mountpoint, path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fmt.Errorf("missing required path: %s", path)
		}
	}

	return nil
}

func GetMounts() ([]string, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var mounts []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			mounts = append(mounts, fields[1])
		}
	}

	return mounts, scanner.Err()
}
