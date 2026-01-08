package ai

import (
	"fmt"
	"strings"

	"github.com/mixos/mix-installer/backend"
	"github.com/mixos/mix-installer/pkg/models"
)

// AutonomousInstaller handles AI-driven autonomous installation
type AutonomousInstaller struct {
	hardware    models.HardwareInfo
	decisions   *DecisionEngine
	recommender *Recommender
	installer   *backend.Installer
	plan        *models.InstallPlan
}

// NewAutonomousInstaller creates a new autonomous installer
func NewAutonomousInstaller() *AutonomousInstaller {
	return &AutonomousInstaller{
		decisions:   NewDecisionEngine(),
		recommender: NewRecommender(),
		installer:   backend.NewInstaller(false),
	}
}

// Analyze analyzes the system and creates an installation plan
func (a *AutonomousInstaller) Analyze() (*models.InstallPlan, error) {
	// Detect hardware
	hw, err := a.detectHardware()
	if err != nil {
		return nil, fmt.Errorf("hardware detection failed: %w", err)
	}
	a.hardware = hw

	// Generate recommendations
	recommendations := a.recommender.GenerateRecommendations(hw)

	// Create installation config based on recommendations
	config := a.createConfig(recommendations)

	// Create plan
	a.plan = &models.InstallPlan{
		Config:          config,
		Recommendations: recommendations,
		EstimatedTime:   a.estimateTime(config),
		RiskLevel:       a.assessRisk(config),
		Warnings:        a.generateWarnings(config),
	}

	return a.plan, nil
}

// Execute executes the autonomous installation
func (a *AutonomousInstaller) Execute() models.InstallResult {
	if a.plan == nil {
		return models.InstallResult{
			Success: false,
			Message: "No installation plan. Run Analyze() first.",
		}
	}

	a.installer.SetConfig(a.plan.Config)
	return a.installer.Install()
}

// GetProgress returns the installation progress channel
func (a *AutonomousInstaller) GetProgress() <-chan models.InstallProgress {
	return a.installer.GetProgress()
}

// detectHardware detects system hardware
func (a *AutonomousInstaller) detectHardware() (models.HardwareInfo, error) {
	hw := models.HardwareInfo{}

	// Detect boot mode
	bootMgr := backend.NewBootloaderManager(true, "")
	hw.BootMode = bootMgr.DetectBootMode()

	// Detect disks
	diskMgr := backend.NewDiskManager(true)
	disks, err := diskMgr.DetectDisks()
	if err != nil {
		return hw, err
	}
	hw.Disks = disks

	// Detect CPU, memory, etc. (simplified)
	hw.CPU = detectCPU()
	hw.Memory = detectMemory()
	hw.Network = detectNetwork()
	hw.GPU = detectGPU()

	return hw, nil
}

// createConfig creates installation config from recommendations
func (a *AutonomousInstaller) createConfig(recommendations []models.AIRecommendation) models.InstallConfig {
	config := models.InstallConfig{
		Mode: models.ModeAutonomous,
	}

	for _, rec := range recommendations {
		switch rec.Type {
		case "disk":
			if disk, ok := rec.Value.(models.DiskConfig); ok {
				config.Disk = disk
			}
		case "partitions":
			if parts, ok := rec.Value.([]models.PartitionConfig); ok {
				config.Partitions = parts
			}
		case "packages":
			if pkgs, ok := rec.Value.(models.PackageConfig); ok {
				config.Packages = pkgs
			}
		case "bootloader":
			if bl, ok := rec.Value.(models.BootloaderConfig); ok {
				config.Bootloader = bl
			}
		case "system":
			if sys, ok := rec.Value.(models.SystemConfig); ok {
				config.System = sys
			}
		}
	}

	// Set defaults if not set
	if config.System.Hostname == "" {
		config.System.Hostname = "mixos"
	}
	if config.System.Timezone == "" {
		config.System.Timezone = "UTC"
	}
	if config.System.Locale == "" {
		config.System.Locale = "en_US.UTF-8"
	}

	// Default packages
	if len(config.Packages.Base) == 0 {
		config.Packages.Base = []string{
			"base", "linux", "linux-firmware",
			"systemd", "networkmanager",
			"docker", "mix-cli", "mix-agent",
		}
	}

	return config
}

// estimateTime estimates installation time in minutes
func (a *AutonomousInstaller) estimateTime(config models.InstallConfig) int {
	baseTime := 5 // Base time in minutes

	// Add time for disk operations
	for _, part := range config.Partitions {
		if part.Format {
			baseTime += 1
		}
	}

	// Add time for packages
	baseTime += len(config.Packages.Base) / 10
	baseTime += len(config.Packages.Additional) / 5

	// Add time for bootloader
	baseTime += 2

	return baseTime
}

// assessRisk assesses the risk level of the installation
func (a *AutonomousInstaller) assessRisk(config models.InstallConfig) string {
	// Check for risky operations
	if config.Disk.WipeAll {
		return "high"
	}

	// Check if installing to removable media
	for _, disk := range a.hardware.Disks {
		if disk.Device == config.Disk.Device && disk.Removable {
			return "medium"
		}
	}

	return "low"
}

// generateWarnings generates warnings for the installation
func (a *AutonomousInstaller) generateWarnings(config models.InstallConfig) []string {
	var warnings []string

	// Check disk size
	for _, disk := range a.hardware.Disks {
		if disk.Device == config.Disk.Device {
			if disk.Size < 20*1024*1024*1024 { // < 20GB
				warnings = append(warnings, "Disk size is less than 20GB. Some features may not work properly.")
			}
		}
	}

	// Check memory
	if a.hardware.Memory.Total < 2*1024*1024*1024 { // < 2GB
		warnings = append(warnings, "System has less than 2GB RAM. Performance may be limited.")
	}

	// Check if wiping disk
	if config.Disk.WipeAll {
		warnings = append(warnings, "All data on the selected disk will be erased!")
	}

	return warnings
}

// SetUserConfig sets user configuration
func (a *AutonomousInstaller) SetUserConfig(user models.UserConfig) {
	if a.plan != nil {
		a.plan.Config.User = user
	}
}

// GetPlan returns the current installation plan
func (a *AutonomousInstaller) GetPlan() *models.InstallPlan {
	return a.plan
}

// ModifyPlan allows modification of the installation plan
func (a *AutonomousInstaller) ModifyPlan(modifier func(*models.InstallPlan)) {
	if a.plan != nil {
		modifier(a.plan)
	}
}

// ValidatePlan validates the installation plan
func (a *AutonomousInstaller) ValidatePlan() []string {
	var errors []string

	if a.plan == nil {
		return []string{"No installation plan"}
	}

	config := a.plan.Config

	// Validate disk
	if config.Disk.Device == "" {
		errors = append(errors, "No disk selected")
	}

	// Validate partitions
	if len(config.Partitions) == 0 {
		errors = append(errors, "No partitions defined")
	}

	hasRoot := false
	for _, part := range config.Partitions {
		if part.MountPoint == "/" {
			hasRoot = true
			break
		}
	}
	if !hasRoot {
		errors = append(errors, "No root partition defined")
	}

	// Validate bootloader for UEFI
	if a.hardware.BootMode == "uefi" {
		hasEFI := false
		for _, part := range config.Partitions {
			if part.MountPoint == "/boot/efi" {
				hasEFI = true
				break
			}
		}
		if !hasEFI {
			errors = append(errors, "UEFI system requires EFI partition")
		}
	}

	return errors
}

// Helper functions for hardware detection

func detectCPU() models.CPUInfo {
	// Simplified CPU detection
	return models.CPUInfo{
		Model:   "Unknown",
		Vendor:  "Unknown",
		Cores:   4,
		Threads: 8,
		Arch:    "x86_64",
	}
}

func detectMemory() models.MemoryInfo {
	// Simplified memory detection
	return models.MemoryInfo{
		Total:     8 * 1024 * 1024 * 1024, // 8GB default
		Available: 6 * 1024 * 1024 * 1024,
	}
}

func detectNetwork() []models.NetworkInfo {
	return []models.NetworkInfo{}
}

func detectGPU() []models.GPUInfo {
	return []models.GPUInfo{}
}

// QuickInstall performs a quick autonomous installation with minimal interaction
func QuickInstall(diskDevice string, username string, password string) models.InstallResult {
	ai := NewAutonomousInstaller()

	// Analyze system
	plan, err := ai.Analyze()
	if err != nil {
		return models.InstallResult{
			Success: false,
			Message: fmt.Sprintf("Analysis failed: %v", err),
		}
	}

	// Override disk if specified
	if diskDevice != "" {
		plan.Config.Disk.Device = diskDevice
		plan.Config.Disk.WipeAll = true
		plan.Config.Disk.UseEntire = true
	}

	// Set user
	if username != "" {
		plan.Config.User = models.UserConfig{
			Username: username,
			Password: password,
			Groups:   []string{"wheel", "docker"},
			Shell:    "/bin/bash",
		}
	}

	// Validate
	if errors := ai.ValidatePlan(); len(errors) > 0 {
		return models.InstallResult{
			Success: false,
			Message: "Validation failed",
			Errors:  errors,
		}
	}

	// Execute
	return ai.Execute()
}

// InteractiveAnalysis provides analysis results for interactive mode
func InteractiveAnalysis() (*models.InstallPlan, []string, error) {
	ai := NewAutonomousInstaller()

	plan, err := ai.Analyze()
	if err != nil {
		return nil, nil, err
	}

	// Generate summary
	var summary []string
	summary = append(summary, fmt.Sprintf("Boot Mode: %s", ai.hardware.BootMode))
	summary = append(summary, fmt.Sprintf("Detected %d disk(s)", len(ai.hardware.Disks)))

	for _, disk := range ai.hardware.Disks {
		summary = append(summary, fmt.Sprintf("  - %s: %s (%s)",
			disk.Device, disk.Model, formatSize(disk.Size)))
	}

	summary = append(summary, fmt.Sprintf("Recommended disk: %s", plan.Config.Disk.Device))
	summary = append(summary, fmt.Sprintf("Estimated time: %d minutes", plan.EstimatedTime))
	summary = append(summary, fmt.Sprintf("Risk level: %s", plan.RiskLevel))

	if len(plan.Warnings) > 0 {
		summary = append(summary, "Warnings:")
		for _, w := range plan.Warnings {
			summary = append(summary, "  ⚠ "+w)
		}
	}

	return plan, summary, nil
}

func formatSize(bytes uint64) string {
	const (
		GB = 1024 * 1024 * 1024
		TB = GB * 1024
	)

	if bytes >= TB {
		return fmt.Sprintf("%.1f TB", float64(bytes)/float64(TB))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
}

// GetHardwareInfo returns detected hardware information
func (a *AutonomousInstaller) GetHardwareInfo() models.HardwareInfo {
	return a.hardware
}

// ExplainDecision explains why a particular decision was made
func (a *AutonomousInstaller) ExplainDecision(decisionType string) string {
	if a.plan == nil {
		return "No plan available"
	}

	for _, rec := range a.plan.Recommendations {
		if rec.Type == decisionType {
			return rec.Reason
		}
	}

	return "Decision not found"
}

// SuggestAlternatives suggests alternatives for a decision
func (a *AutonomousInstaller) SuggestAlternatives(decisionType string) []interface{} {
	if a.plan == nil {
		return nil
	}

	for _, rec := range a.plan.Recommendations {
		if rec.Type == decisionType {
			return rec.Alternatives
		}
	}

	return nil
}

// ApplyAlternative applies an alternative recommendation
func (a *AutonomousInstaller) ApplyAlternative(decisionType string, index int) error {
	if a.plan == nil {
		return fmt.Errorf("no plan available")
	}

	for i, rec := range a.plan.Recommendations {
		if rec.Type == decisionType {
			if index < 0 || index >= len(rec.Alternatives) {
				return fmt.Errorf("invalid alternative index")
			}

			// Update recommendation
			a.plan.Recommendations[i].Value = rec.Alternatives[index]

			// Regenerate config
			a.plan.Config = a.createConfig(a.plan.Recommendations)
			return nil
		}
	}

	return fmt.Errorf("decision type not found: %s", decisionType)
}

// GetInstallationSummary returns a human-readable installation summary
func (a *AutonomousInstaller) GetInstallationSummary() string {
	if a.plan == nil {
		return "No installation plan available"
	}

	var sb strings.Builder

	sb.WriteString("=== MIXOS Installation Summary ===\n\n")

	// Disk
	sb.WriteString(fmt.Sprintf("Target Disk: %s\n", a.plan.Config.Disk.Device))
	sb.WriteString(fmt.Sprintf("  Model: %s\n", a.plan.Config.Disk.Model))
	sb.WriteString(fmt.Sprintf("  Wipe: %v\n\n", a.plan.Config.Disk.WipeAll))

	// Partitions
	sb.WriteString("Partitions:\n")
	for _, part := range a.plan.Config.Partitions {
		sb.WriteString(fmt.Sprintf("  %s -> %s (%s)\n",
			part.Device, part.MountPoint, part.FSType))
	}
	sb.WriteString("\n")

	// System
	sb.WriteString(fmt.Sprintf("Hostname: %s\n", a.plan.Config.System.Hostname))
	sb.WriteString(fmt.Sprintf("Timezone: %s\n", a.plan.Config.System.Timezone))
	sb.WriteString(fmt.Sprintf("Locale: %s\n\n", a.plan.Config.System.Locale))

	// Packages
	sb.WriteString(fmt.Sprintf("Packages: %d base + %d additional\n\n",
		len(a.plan.Config.Packages.Base),
		len(a.plan.Config.Packages.Additional)))

	// Bootloader
	sb.WriteString(fmt.Sprintf("Bootloader: %s (%s)\n\n",
		a.plan.Config.Bootloader.Type,
		a.plan.Config.Bootloader.Target))

	// Estimates
	sb.WriteString(fmt.Sprintf("Estimated Time: %d minutes\n", a.plan.EstimatedTime))
	sb.WriteString(fmt.Sprintf("Risk Level: %s\n", a.plan.RiskLevel))

	return sb.String()
}
