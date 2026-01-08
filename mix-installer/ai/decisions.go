package ai

import (
	"sort"

	"github.com/mixos/mix-installer/pkg/models"
)

// DecisionEngine makes installation decisions based on hardware
type DecisionEngine struct {
	rules []DecisionRule
}

// DecisionRule represents a decision rule
type DecisionRule struct {
	Name      string
	Priority  int
	Condition func(hw models.HardwareInfo) bool
	Action    func(hw models.HardwareInfo) interface{}
}

// NewDecisionEngine creates a new decision engine
func NewDecisionEngine() *DecisionEngine {
	engine := &DecisionEngine{}
	engine.registerDefaultRules()
	return engine
}

// registerDefaultRules registers the default decision rules
func (d *DecisionEngine) registerDefaultRules() {
	// Disk selection rules
	d.rules = append(d.rules, DecisionRule{
		Name:     "prefer_nvme",
		Priority: 100,
		Condition: func(hw models.HardwareInfo) bool {
			for _, disk := range hw.Disks {
				if disk.Type == "nvme" && !disk.Removable && disk.Size > 50*1024*1024*1024 {
					return true
				}
			}
			return false
		},
		Action: func(hw models.HardwareInfo) interface{} {
			for _, disk := range hw.Disks {
				if disk.Type == "nvme" && !disk.Removable && disk.Size > 50*1024*1024*1024 {
					return disk
				}
			}
			return nil
		},
	})

	d.rules = append(d.rules, DecisionRule{
		Name:     "prefer_sata_ssd",
		Priority: 90,
		Condition: func(hw models.HardwareInfo) bool {
			for _, disk := range hw.Disks {
				if disk.Type == "sata" && !disk.Removable && disk.Size > 50*1024*1024*1024 {
					// Check if SSD (simplified - check for no rotation)
					return true
				}
			}
			return false
		},
		Action: func(hw models.HardwareInfo) interface{} {
			for _, disk := range hw.Disks {
				if disk.Type == "sata" && !disk.Removable && disk.Size > 50*1024*1024*1024 {
					return disk
				}
			}
			return nil
		},
	})

	d.rules = append(d.rules, DecisionRule{
		Name:     "fallback_largest_disk",
		Priority: 10,
		Condition: func(hw models.HardwareInfo) bool {
			return len(hw.Disks) > 0
		},
		Action: func(hw models.HardwareInfo) interface{} {
			var largest models.DiskInfo
			for _, disk := range hw.Disks {
				if !disk.Removable && !disk.ReadOnly && disk.Size > largest.Size {
					largest = disk
				}
			}
			return largest
		},
	})

	// Partition scheme rules
	d.rules = append(d.rules, DecisionRule{
		Name:     "uefi_partition_scheme",
		Priority: 100,
		Condition: func(hw models.HardwareInfo) bool {
			return hw.BootMode == "uefi"
		},
		Action: func(hw models.HardwareInfo) interface{} {
			return "gpt_uefi"
		},
	})

	d.rules = append(d.rules, DecisionRule{
		Name:     "legacy_partition_scheme",
		Priority: 100,
		Condition: func(hw models.HardwareInfo) bool {
			return hw.BootMode == "legacy"
		},
		Action: func(hw models.HardwareInfo) interface{} {
			return "mbr_legacy"
		},
	})

	// Swap size rules
	d.rules = append(d.rules, DecisionRule{
		Name:     "swap_for_low_memory",
		Priority: 100,
		Condition: func(hw models.HardwareInfo) bool {
			return hw.Memory.Total < 4*1024*1024*1024 // < 4GB
		},
		Action: func(hw models.HardwareInfo) interface{} {
			// Swap = 2x RAM for low memory systems
			return hw.Memory.Total * 2
		},
	})

	d.rules = append(d.rules, DecisionRule{
		Name:     "swap_for_normal_memory",
		Priority: 90,
		Condition: func(hw models.HardwareInfo) bool {
			return hw.Memory.Total >= 4*1024*1024*1024 && hw.Memory.Total < 16*1024*1024*1024
		},
		Action: func(hw models.HardwareInfo) interface{} {
			// Swap = RAM for normal memory systems
			return hw.Memory.Total
		},
	})

	d.rules = append(d.rules, DecisionRule{
		Name:     "swap_for_high_memory",
		Priority: 80,
		Condition: func(hw models.HardwareInfo) bool {
			return hw.Memory.Total >= 16*1024*1024*1024 // >= 16GB
		},
		Action: func(hw models.HardwareInfo) interface{} {
			// Swap = 8GB max for high memory systems
			return uint64(8 * 1024 * 1024 * 1024)
		},
	})

	// Package selection rules
	d.rules = append(d.rules, DecisionRule{
		Name:     "nvidia_gpu_packages",
		Priority: 100,
		Condition: func(hw models.HardwareInfo) bool {
			for _, gpu := range hw.GPU {
				if gpu.Vendor == "NVIDIA" {
					return true
				}
			}
			return false
		},
		Action: func(hw models.HardwareInfo) interface{} {
			return []string{"nvidia-driver", "nvidia-utils"}
		},
	})

	d.rules = append(d.rules, DecisionRule{
		Name:     "amd_gpu_packages",
		Priority: 100,
		Condition: func(hw models.HardwareInfo) bool {
			for _, gpu := range hw.GPU {
				if gpu.Vendor == "AMD" {
					return true
				}
			}
			return false
		},
		Action: func(hw models.HardwareInfo) interface{} {
			return []string{"mesa", "vulkan-radeon"}
		},
	})

	d.rules = append(d.rules, DecisionRule{
		Name:     "intel_cpu_packages",
		Priority: 100,
		Condition: func(hw models.HardwareInfo) bool {
			return hw.CPU.Vendor == "GenuineIntel"
		},
		Action: func(hw models.HardwareInfo) interface{} {
			return []string{"intel-ucode"}
		},
	})

	d.rules = append(d.rules, DecisionRule{
		Name:     "amd_cpu_packages",
		Priority: 100,
		Condition: func(hw models.HardwareInfo) bool {
			return hw.CPU.Vendor == "AuthenticAMD"
		},
		Action: func(hw models.HardwareInfo) interface{} {
			return []string{"amd-ucode"}
		},
	})
}

// AddRule adds a custom decision rule
func (d *DecisionEngine) AddRule(rule DecisionRule) {
	d.rules = append(d.rules, rule)
}

// Decide makes a decision based on hardware
func (d *DecisionEngine) Decide(hw models.HardwareInfo, decisionType string) interface{} {
	// Sort rules by priority (highest first)
	sortedRules := make([]DecisionRule, len(d.rules))
	copy(sortedRules, d.rules)
	sort.Slice(sortedRules, func(i, j int) bool {
		return sortedRules[i].Priority > sortedRules[j].Priority
	})

	// Find first matching rule
	for _, rule := range sortedRules {
		if rule.Condition(hw) {
			return rule.Action(hw)
		}
	}

	return nil
}

// DecideAll makes all decisions and returns a map
func (d *DecisionEngine) DecideAll(hw models.HardwareInfo) map[string]interface{} {
	decisions := make(map[string]interface{})

	// Sort rules by priority
	sortedRules := make([]DecisionRule, len(d.rules))
	copy(sortedRules, d.rules)
	sort.Slice(sortedRules, func(i, j int) bool {
		return sortedRules[i].Priority > sortedRules[j].Priority
	})

	// Apply all matching rules
	for _, rule := range sortedRules {
		if rule.Condition(hw) {
			decisions[rule.Name] = rule.Action(hw)
		}
	}

	return decisions
}

// SelectDisk selects the best disk for installation
func (d *DecisionEngine) SelectDisk(hw models.HardwareInfo) *models.DiskInfo {
	// Filter out removable and read-only disks
	var candidates []models.DiskInfo
	for _, disk := range hw.Disks {
		if !disk.Removable && !disk.ReadOnly && disk.Size > 20*1024*1024*1024 {
			candidates = append(candidates, disk)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Score each disk
	type scoredDisk struct {
		disk  models.DiskInfo
		score int
	}

	var scored []scoredDisk
	for _, disk := range candidates {
		score := 0

		// Prefer NVMe
		if disk.Type == "nvme" {
			score += 100
		} else if disk.Type == "sata" {
			score += 50
		}

		// Prefer larger disks (up to a point)
		sizeGB := disk.Size / (1024 * 1024 * 1024)
		if sizeGB >= 256 {
			score += 50
		} else if sizeGB >= 128 {
			score += 30
		} else if sizeGB >= 64 {
			score += 10
		}

		// Prefer empty disks
		if len(disk.Partitions) == 0 {
			score += 20
		}

		scored = append(scored, scoredDisk{disk: disk, score: score})
	}

	// Sort by score
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	return &scored[0].disk
}

// CalculatePartitions calculates optimal partition layout
func (d *DecisionEngine) CalculatePartitions(hw models.HardwareInfo, disk models.DiskInfo) []models.PartitionConfig {
	var partitions []models.PartitionConfig
	partNum := 1
	usedSpace := uint64(0)

	// EFI partition for UEFI systems
	if hw.BootMode == "uefi" {
		efiSize := uint64(512 * 1024 * 1024) // 512MB
		partitions = append(partitions, models.PartitionConfig{
			Number:     partNum,
			MountPoint: "/boot/efi",
			Size:       "512MiB",
			SizeBytes:  efiSize,
			FSType:     "fat32",
			Label:      "EFI",
			Flags:      []string{"esp"},
			Format:     true,
		})
		partNum++
		usedSpace += efiSize
	}

	// Calculate swap size
	swapSize := d.calculateSwapSize(hw.Memory.Total)

	// Calculate root size
	remainingSpace := disk.Size - usedSpace - swapSize
	rootSize := remainingSpace

	// If disk is large enough, create separate home partition
	if remainingSpace > 100*1024*1024*1024 { // > 100GB
		rootSize = 50 * 1024 * 1024 * 1024 // 50GB for root
		homeSize := remainingSpace - rootSize

		// Root partition
		partitions = append(partitions, models.PartitionConfig{
			Number:     partNum,
			MountPoint: "/",
			Size:       "50GiB",
			SizeBytes:  rootSize,
			FSType:     "ext4",
			Label:      "MIXOS_ROOT",
			Format:     true,
		})
		partNum++

		// Swap partition
		partitions = append(partitions, models.PartitionConfig{
			Number:     partNum,
			MountPoint: "swap",
			Size:       formatSizeForParted(swapSize),
			SizeBytes:  swapSize,
			FSType:     "linux-swap",
			Label:      "MIXOS_SWAP",
			Flags:      []string{"swap"},
			Format:     true,
		})
		partNum++

		// Home partition
		partitions = append(partitions, models.PartitionConfig{
			Number:     partNum,
			MountPoint: "/home",
			Size:       "100%",
			SizeBytes:  homeSize,
			FSType:     "ext4",
			Label:      "MIXOS_HOME",
			Format:     true,
		})
	} else {
		// Single root partition + swap
		partitions = append(partitions, models.PartitionConfig{
			Number:     partNum,
			MountPoint: "/",
			Size:       formatSizeForParted(rootSize),
			SizeBytes:  rootSize,
			FSType:     "ext4",
			Label:      "MIXOS_ROOT",
			Format:     true,
		})
		partNum++

		partitions = append(partitions, models.PartitionConfig{
			Number:     partNum,
			MountPoint: "swap",
			Size:       "100%",
			SizeBytes:  swapSize,
			FSType:     "linux-swap",
			Label:      "MIXOS_SWAP",
			Flags:      []string{"swap"},
			Format:     true,
		})
	}

	// Assign device names
	for i := range partitions {
		partitions[i].Device = getPartitionDevice(disk.Device, partitions[i].Number)
	}

	return partitions
}

// calculateSwapSize calculates optimal swap size
func (d *DecisionEngine) calculateSwapSize(memoryTotal uint64) uint64 {
	const GB = 1024 * 1024 * 1024

	switch {
	case memoryTotal < 2*GB:
		return memoryTotal * 2 // 2x RAM
	case memoryTotal < 8*GB:
		return memoryTotal // 1x RAM
	case memoryTotal < 64*GB:
		return 8 * GB // 8GB max
	default:
		return 16 * GB // 16GB for very high memory systems
	}
}

// SelectPackages selects packages based on hardware
func (d *DecisionEngine) SelectPackages(hw models.HardwareInfo, profile string) models.PackageConfig {
	config := models.PackageConfig{
		Profile: profile,
		Base: []string{
			"base",
			"linux",
			"linux-firmware",
			"systemd",
			"networkmanager",
			"docker",
			"mix-cli",
			"mix-pkg",
			"mix-agent",
		},
	}

	// Add CPU microcode
	switch hw.CPU.Vendor {
	case "GenuineIntel":
		config.Additional = append(config.Additional, "intel-ucode")
	case "AuthenticAMD":
		config.Additional = append(config.Additional, "amd-ucode")
	}

	// Add GPU drivers
	for _, gpu := range hw.GPU {
		switch gpu.Vendor {
		case "NVIDIA":
			config.Additional = append(config.Additional, "nvidia-driver", "nvidia-utils")
		case "AMD":
			config.Additional = append(config.Additional, "mesa", "vulkan-radeon")
		case "Intel":
			config.Additional = append(config.Additional, "mesa", "vulkan-intel")
		}
	}

	// Add profile-specific packages
	switch profile {
	case "developer":
		config.Additional = append(config.Additional,
			"git", "vim", "tmux",
			"python", "nodejs", "go", "rust",
			"docker-compose",
		)
	case "full":
		config.Additional = append(config.Additional,
			"git", "vim", "tmux",
			"python", "nodejs", "go", "rust",
			"docker-compose",
			"firefox", "code",
		)
	}

	return config
}

// Helper functions

func getPartitionDevice(diskDevice string, partNum int) string {
	// Handle NVMe naming (nvme0n1p1) vs SATA naming (sda1)
	if len(diskDevice) > 0 && diskDevice[len(diskDevice)-1] >= '0' && diskDevice[len(diskDevice)-1] <= '9' {
		return diskDevice + "p" + string('0'+byte(partNum))
	}
	return diskDevice + string('0'+byte(partNum))
}

func formatSizeForParted(bytes uint64) string {
	const (
		MiB = 1024 * 1024
		GiB = MiB * 1024
	)

	if bytes >= GiB {
		return string(rune(bytes/GiB+'0')) + "GiB"
	}
	return string(rune(bytes/MiB+'0')) + "MiB"
}
