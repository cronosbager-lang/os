package ai

import (
	"fmt"

	"github.com/mixos/mix-installer/backend"
	"github.com/mixos/mix-installer/pkg/models"
)

// Recommender generates installation recommendations
type Recommender struct {
	decisions *DecisionEngine
}

// NewRecommender creates a new recommender
func NewRecommender() *Recommender {
	return &Recommender{
		decisions: NewDecisionEngine(),
	}
}

// GenerateRecommendations generates all recommendations for installation
func (r *Recommender) GenerateRecommendations(hw models.HardwareInfo) []models.AIRecommendation {
	var recommendations []models.AIRecommendation

	// Disk recommendation
	diskRec := r.recommendDisk(hw)
	if diskRec != nil {
		recommendations = append(recommendations, *diskRec)
	}

	// Partition recommendation
	if diskRec != nil {
		if disk, ok := diskRec.Value.(models.DiskConfig); ok {
			partRec := r.recommendPartitions(hw, disk)
			if partRec != nil {
				recommendations = append(recommendations, *partRec)
			}
		}
	}

	// Bootloader recommendation
	bootRec := r.recommendBootloader(hw)
	if bootRec != nil {
		recommendations = append(recommendations, *bootRec)
	}

	// Package recommendation
	pkgRec := r.recommendPackages(hw)
	if pkgRec != nil {
		recommendations = append(recommendations, *pkgRec)
	}

	// System configuration recommendation
	sysRec := r.recommendSystemConfig(hw)
	if sysRec != nil {
		recommendations = append(recommendations, *sysRec)
	}

	return recommendations
}

// recommendDisk recommends a disk for installation
func (r *Recommender) recommendDisk(hw models.HardwareInfo) *models.AIRecommendation {
	selectedDisk := r.decisions.SelectDisk(hw)
	if selectedDisk == nil {
		return nil
	}

	// Build alternatives
	var alternatives []interface{}
	for _, disk := range hw.Disks {
		if disk.Device != selectedDisk.Device && !disk.Removable && !disk.ReadOnly {
			alternatives = append(alternatives, models.DiskConfig{
				Device:    disk.Device,
				Model:     disk.Model,
				Size:      disk.Size,
				Type:      disk.Type,
				WipeAll:   true,
				UseEntire: true,
				TableType: r.getTableType(hw),
			})
		}
	}

	reason := r.explainDiskChoice(*selectedDisk, hw)

	return &models.AIRecommendation{
		Type:       "disk",
		Confidence: 0.9,
		Reason:     reason,
		Value: models.DiskConfig{
			Device:    selectedDisk.Device,
			Model:     selectedDisk.Model,
			Size:      selectedDisk.Size,
			Type:      selectedDisk.Type,
			WipeAll:   true,
			UseEntire: true,
			TableType: r.getTableType(hw),
		},
		Alternatives: alternatives,
	}
}

// explainDiskChoice explains why a disk was chosen
func (r *Recommender) explainDiskChoice(disk models.DiskInfo, hw models.HardwareInfo) string {
	reasons := []string{}

	if disk.Type == "nvme" {
		reasons = append(reasons, "NVMe SSD provides fastest performance")
	} else if disk.Type == "sata" {
		reasons = append(reasons, "SATA drive selected")
	}

	sizeGB := disk.Size / (1024 * 1024 * 1024)
	reasons = append(reasons, fmt.Sprintf("%.0f GB capacity is sufficient for MIXOS", float64(sizeGB)))

	if len(disk.Partitions) == 0 {
		reasons = append(reasons, "disk is empty (no existing partitions)")
	}

	if len(reasons) == 0 {
		return "Selected as the best available option"
	}

	result := reasons[0]
	for i := 1; i < len(reasons); i++ {
		result += "; " + reasons[i]
	}
	return result
}

// getTableType returns the recommended partition table type
func (r *Recommender) getTableType(hw models.HardwareInfo) string {
	if hw.BootMode == "uefi" {
		return "gpt"
	}
	return "msdos"
}

// recommendPartitions recommends partition layout
func (r *Recommender) recommendPartitions(hw models.HardwareInfo, disk models.DiskConfig) *models.AIRecommendation {
	// Find the full disk info
	var fullDisk models.DiskInfo
	for _, d := range hw.Disks {
		if d.Device == disk.Device {
			fullDisk = d
			break
		}
	}

	partitions := r.decisions.CalculatePartitions(hw, fullDisk)

	reason := r.explainPartitionChoice(partitions, hw, fullDisk)

	// Generate alternatives
	var alternatives []interface{}

	// Alternative 1: Single partition (no separate home)
	altPartitions := r.generateSinglePartitionLayout(hw, fullDisk)
	if len(altPartitions) != len(partitions) {
		alternatives = append(alternatives, altPartitions)
	}

	// Alternative 2: Btrfs with subvolumes
	btrfsPartitions := r.generateBtrfsLayout(hw, fullDisk)
	alternatives = append(alternatives, btrfsPartitions)

	return &models.AIRecommendation{
		Type:         "partitions",
		Confidence:   0.85,
		Reason:       reason,
		Value:        partitions,
		Alternatives: alternatives,
	}
}

// explainPartitionChoice explains the partition layout
func (r *Recommender) explainPartitionChoice(partitions []models.PartitionConfig, hw models.HardwareInfo, disk models.DiskInfo) string {
	reasons := []string{}

	hasEFI := false
	hasHome := false
	hasSwap := false

	for _, part := range partitions {
		switch part.MountPoint {
		case "/boot/efi":
			hasEFI = true
		case "/home":
			hasHome = true
		case "swap":
			hasSwap = true
		}
	}

	if hasEFI {
		reasons = append(reasons, "EFI partition for UEFI boot")
	}

	if hasHome {
		reasons = append(reasons, "separate /home partition for data protection")
	}

	if hasSwap {
		swapSize := uint64(0)
		for _, part := range partitions {
			if part.MountPoint == "swap" {
				swapSize = part.SizeBytes
				break
			}
		}
		swapGB := float64(swapSize) / (1024 * 1024 * 1024)
		memGB := float64(hw.Memory.Total) / (1024 * 1024 * 1024)
		reasons = append(reasons, fmt.Sprintf("%.1f GB swap (system has %.1f GB RAM)", swapGB, memGB))
	}

	if len(reasons) == 0 {
		return "Standard partition layout"
	}

	result := reasons[0]
	for i := 1; i < len(reasons); i++ {
		result += "; " + reasons[i]
	}
	return result
}

// generateSinglePartitionLayout generates a simple single-partition layout
func (r *Recommender) generateSinglePartitionLayout(hw models.HardwareInfo, disk models.DiskInfo) []models.PartitionConfig {
	var partitions []models.PartitionConfig
	partNum := 1
	usedSpace := uint64(0)

	// EFI partition for UEFI
	if hw.BootMode == "uefi" {
		efiSize := uint64(512 * 1024 * 1024)
		partitions = append(partitions, models.PartitionConfig{
			Number:     partNum,
			Device:     getPartitionDevice(disk.Device, partNum),
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

	// Single root partition
	rootSize := disk.Size - usedSpace
	partitions = append(partitions, models.PartitionConfig{
		Number:     partNum,
		Device:     getPartitionDevice(disk.Device, partNum),
		MountPoint: "/",
		Size:       "100%",
		SizeBytes:  rootSize,
		FSType:     "ext4",
		Label:      "MIXOS_ROOT",
		Format:     true,
	})

	return partitions
}

// generateBtrfsLayout generates a Btrfs layout with subvolumes
func (r *Recommender) generateBtrfsLayout(hw models.HardwareInfo, disk models.DiskInfo) []models.PartitionConfig {
	var partitions []models.PartitionConfig
	partNum := 1
	usedSpace := uint64(0)

	// EFI partition for UEFI
	if hw.BootMode == "uefi" {
		efiSize := uint64(512 * 1024 * 1024)
		partitions = append(partitions, models.PartitionConfig{
			Number:     partNum,
			Device:     getPartitionDevice(disk.Device, partNum),
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

	// Single Btrfs partition with subvolumes
	rootSize := disk.Size - usedSpace
	partitions = append(partitions, models.PartitionConfig{
		Number:     partNum,
		Device:     getPartitionDevice(disk.Device, partNum),
		MountPoint: "/",
		Size:       "100%",
		SizeBytes:  rootSize,
		FSType:     "btrfs",
		Label:      "MIXOS_ROOT",
		Format:     true,
	})

	return partitions
}

// recommendBootloader recommends bootloader configuration
func (r *Recommender) recommendBootloader(hw models.HardwareInfo) *models.AIRecommendation {
	bootMgr := backend.NewBootloaderManager(true, "")
	config := bootMgr.GetRecommendedBootloader()

	reason := "GRUB is the most compatible bootloader"
	if hw.BootMode == "uefi" {
		reason = "GRUB with UEFI support for modern systems"
	}

	// Alternative: systemd-boot for UEFI
	var alternatives []interface{}
	if hw.BootMode == "uefi" {
		alternatives = append(alternatives, models.BootloaderConfig{
			Type:         "systemd-boot",
			Target:       "x86_64-efi",
			EFIDir:       "/boot/efi",
			BootloaderID: "MIXOS",
		})
	}

	return &models.AIRecommendation{
		Type:         "bootloader",
		Confidence:   0.95,
		Reason:       reason,
		Value:        config,
		Alternatives: alternatives,
	}
}

// recommendPackages recommends package selection
func (r *Recommender) recommendPackages(hw models.HardwareInfo) *models.AIRecommendation {
	config := r.decisions.SelectPackages(hw, "developer")

	reason := r.explainPackageChoice(config, hw)

	// Alternatives
	var alternatives []interface{}

	// Minimal profile
	minimalConfig := r.decisions.SelectPackages(hw, "minimal")
	alternatives = append(alternatives, minimalConfig)

	// Full profile
	fullConfig := r.decisions.SelectPackages(hw, "full")
	alternatives = append(alternatives, fullConfig)

	return &models.AIRecommendation{
		Type:         "packages",
		Confidence:   0.8,
		Reason:       reason,
		Value:        config,
		Alternatives: alternatives,
	}
}

// explainPackageChoice explains package selection
func (r *Recommender) explainPackageChoice(config models.PackageConfig, hw models.HardwareInfo) string {
	reasons := []string{
		fmt.Sprintf("Developer profile with %d base packages", len(config.Base)),
	}

	for _, pkg := range config.Additional {
		switch pkg {
		case "intel-ucode":
			reasons = append(reasons, "Intel CPU microcode for security updates")
		case "amd-ucode":
			reasons = append(reasons, "AMD CPU microcode for security updates")
		case "nvidia-driver":
			reasons = append(reasons, "NVIDIA GPU driver for graphics support")
		}
	}

	result := reasons[0]
	for i := 1; i < len(reasons) && i < 3; i++ {
		result += "; " + reasons[i]
	}
	return result
}

// recommendSystemConfig recommends system configuration
func (r *Recommender) recommendSystemConfig(hw models.HardwareInfo) *models.AIRecommendation {
	config := models.SystemConfig{
		Hostname: "mixos-workstation",
		Timezone: "UTC",
		Locale:   "en_US.UTF-8",
		Keymap:   "us",
		Language: "en_US",
	}

	// Try to detect timezone (simplified)
	// In real implementation, would use geolocation or system settings

	return &models.AIRecommendation{
		Type:       "system",
		Confidence: 0.7,
		Reason:     "Default system configuration; customize as needed",
		Value:      config,
		Alternatives: []interface{}{
			models.SystemConfig{
				Hostname: "mixos-server",
				Timezone: "UTC",
				Locale:   "en_US.UTF-8",
				Keymap:   "us",
				Language: "en_US",
			},
		},
	}
}

// GetConfidenceExplanation explains what confidence levels mean
func GetConfidenceExplanation(confidence float64) string {
	switch {
	case confidence >= 0.9:
		return "Very high confidence - strongly recommended"
	case confidence >= 0.8:
		return "High confidence - recommended"
	case confidence >= 0.7:
		return "Moderate confidence - good default"
	case confidence >= 0.5:
		return "Low confidence - consider alternatives"
	default:
		return "Very low confidence - manual review recommended"
	}
}

// ValidateRecommendations validates a set of recommendations
func ValidateRecommendations(recommendations []models.AIRecommendation) []string {
	var errors []string

	hasDisk := false
	hasPartitions := false
	hasBootloader := false

	for _, rec := range recommendations {
		switch rec.Type {
		case "disk":
			hasDisk = true
			if rec.Value == nil {
				errors = append(errors, "No disk selected")
			}
		case "partitions":
			hasPartitions = true
			if parts, ok := rec.Value.([]models.PartitionConfig); ok {
				hasRoot := false
				for _, p := range parts {
					if p.MountPoint == "/" {
						hasRoot = true
						break
					}
				}
				if !hasRoot {
					errors = append(errors, "No root partition in layout")
				}
			}
		case "bootloader":
			hasBootloader = true
		}
	}

	if !hasDisk {
		errors = append(errors, "Missing disk recommendation")
	}
	if !hasPartitions {
		errors = append(errors, "Missing partition recommendation")
	}
	if !hasBootloader {
		errors = append(errors, "Missing bootloader recommendation")
	}

	return errors
}

// ScoreRecommendation scores a recommendation based on various factors
func ScoreRecommendation(rec models.AIRecommendation, hw models.HardwareInfo) float64 {
	baseScore := rec.Confidence

	// Adjust based on type-specific factors
	switch rec.Type {
	case "disk":
		if disk, ok := rec.Value.(models.DiskConfig); ok {
			// Bonus for NVMe
			if disk.Type == "nvme" {
				baseScore += 0.05
			}
			// Bonus for larger disks
			if disk.Size > 256*1024*1024*1024 {
				baseScore += 0.03
			}
		}
	case "partitions":
		if parts, ok := rec.Value.([]models.PartitionConfig); ok {
			// Bonus for separate home
			for _, p := range parts {
				if p.MountPoint == "/home" {
					baseScore += 0.02
					break
				}
			}
		}
	}

	// Cap at 1.0
	if baseScore > 1.0 {
		baseScore = 1.0
	}

	return baseScore
}
