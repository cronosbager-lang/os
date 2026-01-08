package backend

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mixos/mix-installer/pkg/models"
)

// DiskManager handles disk operations
type DiskManager struct {
	dryRun bool
}

// NewDiskManager creates a new disk manager
func NewDiskManager(dryRun bool) *DiskManager {
	return &DiskManager{dryRun: dryRun}
}

// DetectDisks detects all available disks
func (d *DiskManager) DetectDisks() ([]models.DiskInfo, error) {
	var disks []models.DiskInfo

	// Read from /sys/block
	entries, err := os.ReadDir("/sys/block")
	if err != nil {
		return nil, fmt.Errorf("failed to read /sys/block: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip virtual devices
		if strings.HasPrefix(name, "loop") ||
			strings.HasPrefix(name, "ram") ||
			strings.HasPrefix(name, "dm-") ||
			strings.HasPrefix(name, "sr") {
			continue
		}

		disk, err := d.getDiskInfo(name)
		if err != nil {
			continue
		}

		disks = append(disks, disk)
	}

	return disks, nil
}

// getDiskInfo gets detailed information about a disk
func (d *DiskManager) getDiskInfo(name string) (models.DiskInfo, error) {
	disk := models.DiskInfo{
		Device: "/dev/" + name,
	}

	basePath := filepath.Join("/sys/block", name)

	// Get size (in 512-byte sectors)
	if data, err := os.ReadFile(filepath.Join(basePath, "size")); err == nil {
		sectors, _ := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
		disk.Size = sectors * 512
		disk.SizeHuman = formatSize(disk.Size)
	}

	// Get model
	if data, err := os.ReadFile(filepath.Join(basePath, "device/model")); err == nil {
		disk.Model = strings.TrimSpace(string(data))
	}

	// Get serial
	if data, err := os.ReadFile(filepath.Join(basePath, "device/serial")); err == nil {
		disk.Serial = strings.TrimSpace(string(data))
	}

	// Determine type
	if strings.HasPrefix(name, "nvme") {
		disk.Type = "nvme"
		disk.Transport = "nvme"
	} else if strings.HasPrefix(name, "sd") {
		disk.Type = "sata"
		// Check if USB
		if _, err := os.Stat(filepath.Join(basePath, "device/../../driver/usb")); err == nil {
			disk.Transport = "usb"
			disk.Removable = true
		} else {
			disk.Transport = "sata"
		}
	} else if strings.HasPrefix(name, "vd") {
		disk.Type = "virtio"
		disk.Transport = "virtio"
	} else if strings.HasPrefix(name, "hd") {
		disk.Type = "ide"
		disk.Transport = "ide"
	}

	// Check if removable
	if data, err := os.ReadFile(filepath.Join(basePath, "removable")); err == nil {
		disk.Removable = strings.TrimSpace(string(data)) == "1"
	}

	// Check if read-only
	if data, err := os.ReadFile(filepath.Join(basePath, "ro")); err == nil {
		disk.ReadOnly = strings.TrimSpace(string(data)) == "1"
	}

	// Get partitions
	disk.Partitions = d.getPartitions(name)

	return disk, nil
}

// getPartitions gets partition information for a disk
func (d *DiskManager) getPartitions(diskName string) []models.PartitionInfo {
	var partitions []models.PartitionInfo

	// List partitions
	entries, err := os.ReadDir("/sys/block/" + diskName)
	if err != nil {
		return partitions
	}

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, diskName) {
			continue
		}

		part := models.PartitionInfo{
			Device: "/dev/" + name,
		}

		basePath := filepath.Join("/sys/block", diskName, name)

		// Get partition number
		if strings.HasPrefix(diskName, "nvme") {
			// nvme0n1p1 -> 1
			if idx := strings.LastIndex(name, "p"); idx != -1 {
				part.Number, _ = strconv.Atoi(name[idx+1:])
			}
		} else {
			// sda1 -> 1
			for i := len(name) - 1; i >= 0; i-- {
				if name[i] < '0' || name[i] > '9' {
					part.Number, _ = strconv.Atoi(name[i+1:])
					break
				}
			}
		}

		// Get size
		if data, err := os.ReadFile(filepath.Join(basePath, "size")); err == nil {
			sectors, _ := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
			part.Size = sectors * 512
			part.SizeHuman = formatSize(part.Size)
		}

		// Get start sector
		if data, err := os.ReadFile(filepath.Join(basePath, "start")); err == nil {
			part.Start, _ = strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
			part.Start *= 512
		}

		// Get filesystem info using blkid
		if output, err := exec.Command("blkid", "-o", "export", part.Device).Output(); err == nil {
			for _, line := range strings.Split(string(output), "\n") {
				if strings.HasPrefix(line, "TYPE=") {
					part.FSType = strings.TrimPrefix(line, "TYPE=")
				} else if strings.HasPrefix(line, "LABEL=") {
					part.Label = strings.TrimPrefix(line, "LABEL=")
				} else if strings.HasPrefix(line, "UUID=") {
					part.UUID = strings.TrimPrefix(line, "UUID=")
				}
			}
		}

		// Get mount point
		if mounts, err := os.ReadFile("/proc/mounts"); err == nil {
			for _, line := range strings.Split(string(mounts), "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 2 && fields[0] == part.Device {
					part.MountPoint = fields[1]
					break
				}
			}
		}

		partitions = append(partitions, part)
	}

	return partitions
}

// WipeDisk wipes all data from a disk
func (d *DiskManager) WipeDisk(device string) error {
	if d.dryRun {
		fmt.Printf("[DRY RUN] Would wipe disk: %s\n", device)
		return nil
	}

	// Unmount any mounted partitions
	if err := d.unmountAll(device); err != nil {
		return fmt.Errorf("failed to unmount partitions: %w", err)
	}

	// Wipe filesystem signatures
	cmd := exec.Command("wipefs", "-a", device)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("wipefs failed: %w\n%s", err, output)
	}

	// Zero out first and last MB
	if err := d.zeroSectors(device, 0, 2048); err != nil {
		return err
	}

	return nil
}

// unmountAll unmounts all partitions of a disk
func (d *DiskManager) unmountAll(device string) error {
	// Read current mounts
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return err
	}
	defer file.Close()

	var toUnmount []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && strings.HasPrefix(fields[0], device) {
			toUnmount = append(toUnmount, fields[1])
		}
	}

	// Unmount in reverse order (nested mounts)
	for i := len(toUnmount) - 1; i >= 0; i-- {
		cmd := exec.Command("umount", "-l", toUnmount[i])
		cmd.Run() // Ignore errors
	}

	return nil
}

// zeroSectors zeros out sectors on a device
func (d *DiskManager) zeroSectors(device string, start, count int64) error {
	cmd := exec.Command("dd",
		"if=/dev/zero",
		fmt.Sprintf("of=%s", device),
		"bs=512",
		fmt.Sprintf("seek=%d", start),
		fmt.Sprintf("count=%d", count),
		"conv=notrunc",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("dd failed: %w\n%s", err, output)
	}
	return nil
}

// CreatePartitionTable creates a new partition table
func (d *DiskManager) CreatePartitionTable(device string, tableType string) error {
	if d.dryRun {
		fmt.Printf("[DRY RUN] Would create %s partition table on %s\n", tableType, device)
		return nil
	}

	cmd := exec.Command("parted", "-s", device, "mklabel", tableType)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("parted mklabel failed: %w\n%s", err, output)
	}

	return nil
}

// CreatePartition creates a partition
func (d *DiskManager) CreatePartition(device string, config models.PartitionConfig) error {
	if d.dryRun {
		fmt.Printf("[DRY RUN] Would create partition on %s: %+v\n", device, config)
		return nil
	}

	// Build parted command
	args := []string{"-s", "-a", "optimal", device, "mkpart"}

	// Partition type (for MBR)
	if config.FSType == "fat32" || config.FSType == "vfat" {
		args = append(args, "primary", "fat32")
	} else if config.FSType == "linux-swap" {
		args = append(args, "primary", "linux-swap")
	} else {
		args = append(args, "primary", "ext4")
	}

	// Start and end
	args = append(args, config.Size) // parted handles size strings

	cmd := exec.Command("parted", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("parted mkpart failed: %w\n%s", err, output)
	}

	// Set flags
	for _, flag := range config.Flags {
		flagCmd := exec.Command("parted", "-s", device, "set", fmt.Sprintf("%d", config.Number), flag, "on")
		if output, err := flagCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("parted set flag failed: %w\n%s", err, output)
		}
	}

	// Wait for device to appear
	exec.Command("partprobe", device).Run()
	exec.Command("udevadm", "settle").Run()

	return nil
}

// CreatePartitions creates all partitions according to config
func (d *DiskManager) CreatePartitions(device string, partitions []models.PartitionConfig) error {
	for _, part := range partitions {
		if err := d.CreatePartition(device, part); err != nil {
			return fmt.Errorf("failed to create partition %d: %w", part.Number, err)
		}
	}
	return nil
}

// GetRecommendedPartitions returns recommended partition layout
func (d *DiskManager) GetRecommendedPartitions(disk models.DiskInfo, bootMode string) []models.PartitionConfig {
	var partitions []models.PartitionConfig

	diskSize := disk.Size
	partNum := 1

	// EFI partition (for UEFI boot)
	if bootMode == "uefi" {
		partitions = append(partitions, models.PartitionConfig{
			Number:     partNum,
			MountPoint: "/boot/efi",
			Size:       "512MiB",
			SizeBytes:  512 * 1024 * 1024,
			FSType:     "fat32",
			Label:      "EFI",
			Flags:      []string{"esp"},
			Format:     true,
		})
		partNum++
		diskSize -= 512 * 1024 * 1024
	}

	// Calculate swap size (RAM size, max 8GB)
	memInfo := getMemorySize()
	swapSize := memInfo
	if swapSize > 8*1024*1024*1024 {
		swapSize = 8 * 1024 * 1024 * 1024
	}

	// Root partition
	rootSize := diskSize - swapSize
	if rootSize > 100*1024*1024*1024 { // If > 100GB, limit root and create home
		rootSize = 50 * 1024 * 1024 * 1024 // 50GB for root
	}

	partitions = append(partitions, models.PartitionConfig{
		Number:     partNum,
		MountPoint: "/",
		Size:       formatSize(uint64(rootSize)),
		SizeBytes:  uint64(rootSize),
		FSType:     "ext4",
		Label:      "MIXOS_ROOT",
		Format:     true,
	})
	partNum++

	// Swap partition
	partitions = append(partitions, models.PartitionConfig{
		Number:     partNum,
		MountPoint: "swap",
		Size:       formatSize(uint64(swapSize)),
		SizeBytes:  uint64(swapSize),
		FSType:     "linux-swap",
		Label:      "MIXOS_SWAP",
		Flags:      []string{"swap"},
		Format:     true,
	})
	partNum++

	// Home partition (if space available)
	remainingSize := diskSize - uint64(rootSize) - uint64(swapSize)
	if remainingSize > 10*1024*1024*1024 { // > 10GB
		partitions = append(partitions, models.PartitionConfig{
			Number:     partNum,
			MountPoint: "/home",
			Size:       "100%",
			SizeBytes:  remainingSize,
			FSType:     "ext4",
			Label:      "MIXOS_HOME",
			Format:     true,
		})
	}

	return partitions
}

// getMemorySize returns total system memory in bytes
func getMemorySize() uint64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 4 * 1024 * 1024 * 1024 // Default 4GB
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, _ := strconv.ParseUint(fields[1], 10, 64)
				return kb * 1024
			}
		}
	}

	return 4 * 1024 * 1024 * 1024
}

// formatSize formats bytes to human readable string
func formatSize(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1fTiB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.1fGiB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fMiB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fKiB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
