package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mixos/mix-installer/pkg/models"
)

// Installer orchestrates the installation process
type Installer struct {
	config     models.InstallConfig
	disk       *DiskManager
	fs         *FilesystemManager
	bootloader *BootloaderManager
	packages   *PackageManager
	dryRun     bool
	mountRoot  string
	progress   chan models.InstallProgress
	startTime  time.Time
}

// NewInstaller creates a new installer
func NewInstaller(dryRun bool) *Installer {
	mountRoot := "/mnt"
	return &Installer{
		disk:       NewDiskManager(dryRun),
		fs:         NewFilesystemManager(dryRun, mountRoot),
		bootloader: NewBootloaderManager(dryRun, mountRoot),
		packages:   NewPackageManager(dryRun, mountRoot),
		dryRun:     dryRun,
		mountRoot:  mountRoot,
		progress:   make(chan models.InstallProgress, 100),
	}
}

// SetConfig sets the installation configuration
func (i *Installer) SetConfig(config models.InstallConfig) {
	i.config = config
}

// GetProgress returns the progress channel
func (i *Installer) GetProgress() <-chan models.InstallProgress {
	return i.progress
}

// Install performs the complete installation
func (i *Installer) Install() models.InstallResult {
	i.startTime = time.Now()
	result := models.InstallResult{
		Success: true,
	}

	phases := []struct {
		name string
		fn   func() error
	}{
		{"Preparing disk", i.prepareDisk},
		{"Creating partitions", i.createPartitions},
		{"Formatting partitions", i.formatPartitions},
		{"Mounting filesystems", i.mountFilesystems},
		{"Installing base system", i.installBaseSystem},
		{"Installing packages", i.installPackages},
		{"Configuring system", i.configureSystem},
		{"Creating user", i.createUser},
		{"Installing bootloader", i.installBootloader},
		{"Finalizing", i.finalize},
	}

	totalPhases := len(phases)
	for idx, phase := range phases {
		progress := float64(idx) / float64(totalPhases) * 100

		i.sendProgress(phase.name, "", progress, "")

		if err := phase.fn(); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", phase.name, err))
			i.sendProgress(phase.name, "Failed", progress, err.Error())
			break
		}

		i.sendProgress(phase.name, "Complete", float64(idx+1)/float64(totalPhases)*100, "")
	}

	result.Duration = int64(time.Since(i.startTime).Seconds())

	if result.Success {
		result.Message = "Installation completed successfully"
		result.RebootReady = true
	} else {
		result.Message = "Installation failed"
	}

	close(i.progress)
	return result
}

// sendProgress sends a progress update
func (i *Installer) sendProgress(phase, step string, progress float64, errMsg string) {
	select {
	case i.progress <- models.InstallProgress{
		Phase:     phase,
		Step:      step,
		Progress:  progress,
		Message:   fmt.Sprintf("%s: %s", phase, step),
		Error:     errMsg,
		StartedAt: i.startTime.Unix(),
	}:
	default:
	}
}

// prepareDisk prepares the disk for installation
func (i *Installer) prepareDisk() error {
	if i.config.Disk.WipeAll {
		i.sendProgress("Preparing disk", "Wiping disk", 5, "")
		if err := i.disk.WipeDisk(i.config.Disk.Device); err != nil {
			return err
		}
	}

	i.sendProgress("Preparing disk", "Creating partition table", 10, "")
	tableType := i.config.Disk.TableType
	if tableType == "" {
		if i.bootloader.DetectBootMode() == "uefi" {
			tableType = "gpt"
		} else {
			tableType = "msdos"
		}
	}

	return i.disk.CreatePartitionTable(i.config.Disk.Device, tableType)
}

// createPartitions creates the partitions
func (i *Installer) createPartitions() error {
	for idx, part := range i.config.Partitions {
		i.sendProgress("Creating partitions", fmt.Sprintf("Partition %d", idx+1), 15+float64(idx)*2, "")
		if err := i.disk.CreatePartition(i.config.Disk.Device, part); err != nil {
			return err
		}
	}
	return nil
}

// formatPartitions formats the partitions
func (i *Installer) formatPartitions() error {
	for idx, part := range i.config.Partitions {
		if !part.Format {
			continue
		}
		i.sendProgress("Formatting partitions", fmt.Sprintf("Formatting %s", part.Device), 25+float64(idx)*2, "")
		if err := i.fs.FormatPartition(part.Device, part.FSType, part.Label); err != nil {
			return err
		}
	}
	return nil
}

// mountFilesystems mounts the filesystems
func (i *Installer) mountFilesystems() error {
	i.sendProgress("Mounting filesystems", "Mounting partitions", 35, "")

	if err := i.fs.MountPartitions(i.config.Partitions); err != nil {
		return err
	}

	// Enable swap
	for _, part := range i.config.Partitions {
		if part.MountPoint == "swap" {
			if err := i.fs.EnableSwap(part.Device); err != nil {
				return err
			}
		}
	}

	return nil
}

// installBaseSystem installs the base system
func (i *Installer) installBaseSystem() error {
	i.sendProgress("Installing base system", "Extracting rootfs", 40, "")

	// Check for squashfs image
	squashfsPath := "/run/mixos/rootfs.sfs"
	if _, err := os.Stat(squashfsPath); err == nil {
		return i.extractSquashfs(squashfsPath)
	}

	// Fallback to pacstrap-like installation
	return i.bootstrapSystem()
}

// extractSquashfs extracts the squashfs image
func (i *Installer) extractSquashfs(path string) error {
	if i.dryRun {
		fmt.Printf("[DRY RUN] Would extract %s to %s\n", path, i.mountRoot)
		return nil
	}

	cmd := exec.Command("unsquashfs", "-f", "-d", i.mountRoot, path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unsquashfs failed: %w\n%s", err, output)
	}

	return nil
}

// bootstrapSystem bootstraps a minimal system
func (i *Installer) bootstrapSystem() error {
	if i.dryRun {
		fmt.Printf("[DRY RUN] Would bootstrap system to %s\n", i.mountRoot)
		return nil
	}

	// Create essential directories
	dirs := []string{
		"bin", "sbin", "lib", "lib64",
		"usr/bin", "usr/sbin", "usr/lib", "usr/lib64",
		"etc", "var", "tmp", "root", "home",
		"proc", "sys", "dev", "run",
		"boot", "opt", "mnt",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(i.mountRoot, dir), 0755); err != nil {
			return err
		}
	}

	// Copy essential files from live system
	essentialFiles := []string{
		"/etc/os-release",
		"/etc/passwd",
		"/etc/group",
		"/etc/shadow",
	}

	for _, file := range essentialFiles {
		if data, err := os.ReadFile(file); err == nil {
			dstPath := filepath.Join(i.mountRoot, file)
			os.MkdirAll(filepath.Dir(dstPath), 0755)
			os.WriteFile(dstPath, data, 0644)
		}
	}

	return nil
}

// installPackages installs selected packages
func (i *Installer) installPackages() error {
	i.sendProgress("Installing packages", "Installing base packages", 50, "")

	packages := i.config.Packages.Base
	packages = append(packages, i.config.Packages.Additional...)

	// Remove excluded packages
	excludeMap := make(map[string]bool)
	for _, pkg := range i.config.Packages.Exclude {
		excludeMap[pkg] = true
	}

	var filtered []string
	for _, pkg := range packages {
		if !excludeMap[pkg] {
			filtered = append(filtered, pkg)
		}
	}

	return i.packages.InstallPackages(filtered)
}

// configureSystem configures the installed system
func (i *Installer) configureSystem() error {
	i.sendProgress("Configuring system", "Setting hostname", 70, "")

	// Hostname
	if err := i.writeFile("/etc/hostname", i.config.System.Hostname+"\n"); err != nil {
		return err
	}

	// Hosts
	hosts := fmt.Sprintf("127.0.0.1   localhost\n127.0.1.1   %s\n::1         localhost\n",
		i.config.System.Hostname)
	if err := i.writeFile("/etc/hosts", hosts); err != nil {
		return err
	}

	i.sendProgress("Configuring system", "Setting timezone", 72, "")

	// Timezone
	if i.config.System.Timezone != "" {
		tzPath := filepath.Join(i.mountRoot, "etc", "localtime")
		os.Remove(tzPath)
		if !i.dryRun {
			os.Symlink(
				filepath.Join("/usr/share/zoneinfo", i.config.System.Timezone),
				tzPath,
			)
		}
	}

	i.sendProgress("Configuring system", "Setting locale", 74, "")

	// Locale
	if i.config.System.Locale != "" {
		localeGen := i.config.System.Locale + " UTF-8\n"
		if err := i.writeFile("/etc/locale.gen", localeGen); err != nil {
			return err
		}

		localeConf := fmt.Sprintf("LANG=%s\n", i.config.System.Locale)
		if err := i.writeFile("/etc/locale.conf", localeConf); err != nil {
			return err
		}

		// Generate locales
		if !i.dryRun {
			exec.Command("chroot", i.mountRoot, "locale-gen").Run()
		}
	}

	i.sendProgress("Configuring system", "Writing fstab", 76, "")

	// Fstab
	if err := i.fs.WriteFstab(i.config.Partitions); err != nil {
		return err
	}

	return nil
}

// createUser creates the user account
func (i *Installer) createUser() error {
	i.sendProgress("Creating user", "Creating user account", 80, "")

	if i.config.User.Username == "" {
		return nil
	}

	if i.dryRun {
		fmt.Printf("[DRY RUN] Would create user: %s\n", i.config.User.Username)
		return nil
	}

	// Create user
	args := []string{
		i.mountRoot,
		"useradd",
		"-m",
		"-s", i.config.User.Shell,
	}

	if len(i.config.User.Groups) > 0 {
		args = append(args, "-G", strings.Join(i.config.User.Groups, ","))
	}

	if i.config.User.FullName != "" {
		args = append(args, "-c", i.config.User.FullName)
	}

	args = append(args, i.config.User.Username)

	cmd := exec.Command("chroot", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("useradd failed: %w\n%s", err, output)
	}

	i.sendProgress("Creating user", "Setting password", 82, "")

	// Set password
	if i.config.User.Password != "" {
		cmd := exec.Command("chroot", i.mountRoot, "chpasswd")
		cmd.Stdin = strings.NewReader(fmt.Sprintf("%s:%s\n",
			i.config.User.Username, i.config.User.Password))
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("chpasswd failed: %w\n%s", err, output)
		}
	}

	// Set root password
	if i.config.User.RootPassword != "" {
		cmd := exec.Command("chroot", i.mountRoot, "chpasswd")
		cmd.Stdin = strings.NewReader(fmt.Sprintf("root:%s\n", i.config.User.RootPassword))
		cmd.Run()
	}

	return nil
}

// installBootloader installs the bootloader
func (i *Installer) installBootloader() error {
	i.sendProgress("Installing bootloader", "Installing GRUB", 85, "")

	// Write GRUB defaults
	if err := i.bootloader.WriteGRUBDefaults(); err != nil {
		return err
	}

	// Install GRUB
	if err := i.bootloader.InstallGRUB(i.config.Bootloader); err != nil {
		return err
	}

	i.sendProgress("Installing bootloader", "Generating initramfs", 90, "")

	// Update initramfs
	if err := i.bootloader.UpdateInitramfs(); err != nil {
		// Non-fatal, just warn
		fmt.Printf("Warning: initramfs update failed: %v\n", err)
	}

	// Setup EFI fallback
	if i.bootloader.DetectBootMode() == "uefi" {
		i.bootloader.SetupEFIFallback(i.config.Bootloader)
	}

	return nil
}

// finalize performs final cleanup
func (i *Installer) finalize() error {
	i.sendProgress("Finalizing", "Enabling services", 95, "")

	// Enable essential services
	services := []string{
		"systemd-networkd",
		"systemd-resolved",
		"docker",
		"mixos-agent",
	}

	for _, svc := range services {
		if !i.dryRun {
			exec.Command("chroot", i.mountRoot, "systemctl", "enable", svc).Run()
		}
	}

	i.sendProgress("Finalizing", "Syncing filesystems", 98, "")

	// Sync
	exec.Command("sync").Run()

	i.sendProgress("Finalizing", "Unmounting filesystems", 99, "")

	// Disable swap
	for _, part := range i.config.Partitions {
		if part.MountPoint == "swap" {
			i.fs.DisableSwap(part.Device)
		}
	}

	// Unmount
	i.fs.UnmountAll()

	return nil
}

// writeFile writes a file to the installed system
func (i *Installer) writeFile(path string, content string) error {
	fullPath := filepath.Join(i.mountRoot, path)

	if i.dryRun {
		fmt.Printf("[DRY RUN] Would write to %s:\n%s\n", fullPath, content)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, []byte(content), 0644)
}

// Cleanup cleans up after installation (on failure)
func (i *Installer) Cleanup() {
	i.fs.UnmountAll()
}

// GetMountRoot returns the mount root path
func (i *Installer) GetMountRoot() string {
	return i.mountRoot
}

// PackageManager handles package installation
type PackageManager struct {
	dryRun    bool
	mountRoot string
}

// NewPackageManager creates a new package manager
func NewPackageManager(dryRun bool, mountRoot string) *PackageManager {
	return &PackageManager{
		dryRun:    dryRun,
		mountRoot: mountRoot,
	}
}

// InstallPackages installs packages to the target system
func (p *PackageManager) InstallPackages(packages []string) error {
	if len(packages) == 0 {
		return nil
	}

	if p.dryRun {
		fmt.Printf("[DRY RUN] Would install packages: %v\n", packages)
		return nil
	}

	// Try mix-pkg first
	if _, err := exec.LookPath("mix-pkg"); err == nil {
		args := append([]string{p.mountRoot, "mix-pkg", "install", "-y"}, packages...)
		cmd := exec.Command("chroot", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("mix-pkg install failed: %w\n%s", err, output)
		}
		return nil
	}

	// Fallback to pacman
	if _, err := exec.LookPath("pacman"); err == nil {
		args := append([]string{"-r", p.mountRoot, "-S", "--noconfirm"}, packages...)
		cmd := exec.Command("pacman", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("pacman install failed: %w\n%s", err, output)
		}
		return nil
	}

	return fmt.Errorf("no package manager available")
}
