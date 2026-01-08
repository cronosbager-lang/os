package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mixos/mix-installer/pkg/models"
)

// FilesystemManager handles filesystem operations
type FilesystemManager struct {
	dryRun    bool
	mountRoot string
}

// NewFilesystemManager creates a new filesystem manager
func NewFilesystemManager(dryRun bool, mountRoot string) *FilesystemManager {
	if mountRoot == "" {
		mountRoot = "/mnt"
	}
	return &FilesystemManager{
		dryRun:    dryRun,
		mountRoot: mountRoot,
	}
}

// FormatPartition formats a partition with the specified filesystem
func (f *FilesystemManager) FormatPartition(device string, fsType string, label string) error {
	if f.dryRun {
		fmt.Printf("[DRY RUN] Would format %s as %s (label: %s)\n", device, fsType, label)
		return nil
	}

	var cmd *exec.Cmd

	switch fsType {
	case "ext4":
		args := []string{"-F", "-t", "ext4"}
		if label != "" {
			args = append(args, "-L", label)
		}
		args = append(args, device)
		cmd = exec.Command("mkfs.ext4", args...)

	case "ext3":
		args := []string{"-F", "-t", "ext3"}
		if label != "" {
			args = append(args, "-L", label)
		}
		args = append(args, device)
		cmd = exec.Command("mkfs.ext3", args...)

	case "btrfs":
		args := []string{"-f"}
		if label != "" {
			args = append(args, "-L", label)
		}
		args = append(args, device)
		cmd = exec.Command("mkfs.btrfs", args...)

	case "xfs":
		args := []string{"-f"}
		if label != "" {
			args = append(args, "-L", label)
		}
		args = append(args, device)
		cmd = exec.Command("mkfs.xfs", args...)

	case "fat32", "vfat":
		args := []string{"-F", "32"}
		if label != "" {
			args = append(args, "-n", label)
		}
		args = append(args, device)
		cmd = exec.Command("mkfs.fat", args...)

	case "linux-swap", "swap":
		args := []string{}
		if label != "" {
			args = append(args, "-L", label)
		}
		args = append(args, device)
		cmd = exec.Command("mkswap", args...)

	default:
		return fmt.Errorf("unsupported filesystem type: %s", fsType)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("format failed: %w\n%s", err, output)
	}

	return nil
}

// FormatPartitions formats all partitions according to config
func (f *FilesystemManager) FormatPartitions(partitions []models.PartitionConfig) error {
	for _, part := range partitions {
		if !part.Format {
			continue
		}

		if err := f.FormatPartition(part.Device, part.FSType, part.Label); err != nil {
			return fmt.Errorf("failed to format %s: %w", part.Device, err)
		}
	}
	return nil
}

// MountPartition mounts a partition
func (f *FilesystemManager) MountPartition(device string, mountPoint string, fsType string, options string) error {
	fullPath := filepath.Join(f.mountRoot, mountPoint)

	if f.dryRun {
		fmt.Printf("[DRY RUN] Would mount %s on %s (type: %s)\n", device, fullPath, fsType)
		return nil
	}

	// Create mount point
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	// Build mount command
	args := []string{}
	if fsType != "" {
		args = append(args, "-t", fsType)
	}
	if options != "" {
		args = append(args, "-o", options)
	}
	args = append(args, device, fullPath)

	cmd := exec.Command("mount", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount failed: %w\n%s", err, output)
	}

	return nil
}

// MountPartitions mounts all partitions according to config
func (f *FilesystemManager) MountPartitions(partitions []models.PartitionConfig) error {
	// Sort by mount point depth (/ first, then /boot, /home, etc.)
	sorted := sortByMountPoint(partitions)

	for _, part := range sorted {
		if part.MountPoint == "" || part.MountPoint == "swap" {
			continue
		}

		fsType := part.FSType
		if fsType == "fat32" {
			fsType = "vfat"
		}

		if err := f.MountPartition(part.Device, part.MountPoint, fsType, ""); err != nil {
			return fmt.Errorf("failed to mount %s: %w", part.Device, err)
		}
	}

	return nil
}

// UnmountAll unmounts all filesystems under mount root
func (f *FilesystemManager) UnmountAll() error {
	if f.dryRun {
		fmt.Printf("[DRY RUN] Would unmount all under %s\n", f.mountRoot)
		return nil
	}

	// Get list of mounts under mountRoot
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return err
	}

	var mounts []string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.HasPrefix(fields[1], f.mountRoot) {
			mounts = append(mounts, fields[1])
		}
	}

	// Unmount in reverse order (deepest first)
	for i := len(mounts) - 1; i >= 0; i-- {
		cmd := exec.Command("umount", "-l", mounts[i])
		cmd.Run() // Ignore errors
	}

	return nil
}

// EnableSwap enables swap on a partition
func (f *FilesystemManager) EnableSwap(device string) error {
	if f.dryRun {
		fmt.Printf("[DRY RUN] Would enable swap on %s\n", device)
		return nil
	}

	cmd := exec.Command("swapon", device)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("swapon failed: %w\n%s", err, output)
	}

	return nil
}

// DisableSwap disables swap on a partition
func (f *FilesystemManager) DisableSwap(device string) error {
	if f.dryRun {
		fmt.Printf("[DRY RUN] Would disable swap on %s\n", device)
		return nil
	}

	cmd := exec.Command("swapoff", device)
	cmd.Run() // Ignore errors
	return nil
}

// GenerateFstab generates /etc/fstab content
func (f *FilesystemManager) GenerateFstab(partitions []models.PartitionConfig) (string, error) {
	var lines []string

	lines = append(lines, "# /etc/fstab: static file system information")
	lines = append(lines, "# <device>  <mount>  <type>  <options>  <dump>  <pass>")
	lines = append(lines, "")

	for _, part := range partitions {
		if part.MountPoint == "" {
			continue
		}

		// Get UUID
		uuid := f.getUUID(part.Device)
		if uuid == "" {
			uuid = part.Device
		} else {
			uuid = "UUID=" + uuid
		}

		fsType := part.FSType
		if fsType == "fat32" {
			fsType = "vfat"
		}

		var options string
		var dump, pass int

		switch part.MountPoint {
		case "/":
			options = "defaults"
			dump = 0
			pass = 1
		case "/boot/efi":
			options = "umask=0077"
			dump = 0
			pass = 2
		case "swap":
			options = "defaults"
			dump = 0
			pass = 0
		default:
			options = "defaults"
			dump = 0
			pass = 2
		}

		mountPoint := part.MountPoint
		if mountPoint == "swap" {
			mountPoint = "none"
			fsType = "swap"
		}

		line := fmt.Sprintf("%-40s %-15s %-8s %-15s %d %d",
			uuid, mountPoint, fsType, options, dump, pass)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n") + "\n", nil
}

// WriteFstab writes the fstab file
func (f *FilesystemManager) WriteFstab(partitions []models.PartitionConfig) error {
	content, err := f.GenerateFstab(partitions)
	if err != nil {
		return err
	}

	fstabPath := filepath.Join(f.mountRoot, "etc", "fstab")

	if f.dryRun {
		fmt.Printf("[DRY RUN] Would write fstab to %s:\n%s\n", fstabPath, content)
		return nil
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fstabPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(fstabPath, []byte(content), 0644)
}

// getUUID gets the UUID of a device
func (f *FilesystemManager) getUUID(device string) string {
	output, err := exec.Command("blkid", "-s", "UUID", "-o", "value", device).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// sortByMountPoint sorts partitions by mount point depth
func sortByMountPoint(partitions []models.PartitionConfig) []models.PartitionConfig {
	sorted := make([]models.PartitionConfig, len(partitions))
	copy(sorted, partitions)

	// Simple bubble sort by path depth
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			depth1 := strings.Count(sorted[j].MountPoint, "/")
			depth2 := strings.Count(sorted[j+1].MountPoint, "/")
			if depth1 > depth2 {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}

// GetMountRoot returns the mount root path
func (f *FilesystemManager) GetMountRoot() string {
	return f.mountRoot
}

// CheckFilesystemSupport checks if a filesystem type is supported
func (f *FilesystemManager) CheckFilesystemSupport(fsType string) bool {
	switch fsType {
	case "ext4", "ext3", "ext2":
		_, err := exec.LookPath("mkfs.ext4")
		return err == nil
	case "btrfs":
		_, err := exec.LookPath("mkfs.btrfs")
		return err == nil
	case "xfs":
		_, err := exec.LookPath("mkfs.xfs")
		return err == nil
	case "fat32", "vfat":
		_, err := exec.LookPath("mkfs.fat")
		return err == nil
	case "linux-swap", "swap":
		_, err := exec.LookPath("mkswap")
		return err == nil
	default:
		return false
	}
}
