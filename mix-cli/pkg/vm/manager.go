package vm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Manager handles VM operations
type Manager struct {
	vmDir    string
	backend  VMBackend
}

// VMBackend represents the virtualization backend
type VMBackend string

const (
	BackendQEMU   VMBackend = "qemu"
	BackendLibvirt VMBackend = "libvirt"
)

// VM represents a virtual machine
type VM struct {
	Name     string            `json:"name"`
	State    string            `json:"state"`
	CPUs     int               `json:"cpus"`
	Memory   int               `json:"memory"` // MB
	DiskSize int               `json:"disk_size"` // GB
	DiskPath string            `json:"disk_path"`
	ISO      string            `json:"iso,omitempty"`
	Network  string            `json:"network"`
	VNC      int               `json:"vnc_port,omitempty"`
	SSH      int               `json:"ssh_port,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// VMConfig represents VM creation configuration
type VMConfig struct {
	Name     string
	CPUs     int
	Memory   int // MB
	DiskSize int // GB
	ISO      string
	Network  string
	OS       string
}

// NewManager creates a new VM manager
func NewManager() *Manager {
	vmDir := filepath.Join(os.Getenv("HOME"), ".mixos/vms")
	os.MkdirAll(vmDir, 0755)

	backend := BackendQEMU
	if _, err := exec.LookPath("virsh"); err == nil {
		backend = BackendLibvirt
	}

	return &Manager{
		vmDir:   vmDir,
		backend: backend,
	}
}

// Create creates a new VM
func (m *Manager) Create(config VMConfig) (*VM, error) {
	// Validate config
	if config.Name == "" {
		return nil, fmt.Errorf("VM name is required")
	}
	if config.CPUs <= 0 {
		config.CPUs = 2
	}
	if config.Memory <= 0 {
		config.Memory = 2048
	}
	if config.DiskSize <= 0 {
		config.DiskSize = 20
	}
	if config.Network == "" {
		config.Network = "user"
	}

	// Create VM directory
	vmPath := filepath.Join(m.vmDir, config.Name)
	if err := os.MkdirAll(vmPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create VM directory: %w", err)
	}

	// Create disk image
	diskPath := filepath.Join(vmPath, "disk.qcow2")
	if err := m.createDisk(diskPath, config.DiskSize); err != nil {
		return nil, fmt.Errorf("failed to create disk: %w", err)
	}

	vm := &VM{
		Name:     config.Name,
		State:    "stopped",
		CPUs:     config.CPUs,
		Memory:   config.Memory,
		DiskSize: config.DiskSize,
		DiskPath: diskPath,
		ISO:      config.ISO,
		Network:  config.Network,
		Metadata: make(map[string]string),
	}

	// Save VM config
	if err := m.saveVM(vm); err != nil {
		return nil, fmt.Errorf("failed to save VM config: %w", err)
	}

	return vm, nil
}

// Start starts a VM
func (m *Manager) Start(name string) error {
	vm, err := m.Get(name)
	if err != nil {
		return err
	}

	if vm.State == "running" {
		return fmt.Errorf("VM is already running")
	}

	// Build QEMU command
	args := []string{
		"-name", vm.Name,
		"-m", fmt.Sprintf("%d", vm.Memory),
		"-smp", fmt.Sprintf("%d", vm.CPUs),
		"-drive", fmt.Sprintf("file=%s,format=qcow2", vm.DiskPath),
		"-enable-kvm",
		"-daemonize",
	}

	// Add ISO if specified
	if vm.ISO != "" {
		args = append(args, "-cdrom", vm.ISO)
		args = append(args, "-boot", "d")
	}

	// Network configuration
	switch vm.Network {
	case "user":
		// Find available port for SSH forwarding
		sshPort := findAvailablePort(2222)
		args = append(args, "-netdev", fmt.Sprintf("user,id=net0,hostfwd=tcp::%d-:22", sshPort))
		args = append(args, "-device", "virtio-net-pci,netdev=net0")
		vm.SSH = sshPort
	case "bridge":
		args = append(args, "-netdev", "bridge,id=net0,br=br0")
		args = append(args, "-device", "virtio-net-pci,netdev=net0")
	}

	// VNC
	vncPort := findAvailablePort(5900)
	args = append(args, "-vnc", fmt.Sprintf(":%d", vncPort-5900))
	vm.VNC = vncPort

	// PID file
	pidFile := filepath.Join(m.vmDir, vm.Name, "qemu.pid")
	args = append(args, "-pidfile", pidFile)

	// Monitor socket
	monitorSocket := filepath.Join(m.vmDir, vm.Name, "monitor.sock")
	args = append(args, "-monitor", fmt.Sprintf("unix:%s,server,nowait", monitorSocket))

	// Run QEMU
	cmd := exec.Command("qemu-system-x86_64", args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}

	vm.State = "running"
	return m.saveVM(vm)
}

// Stop stops a VM
func (m *Manager) Stop(name string, force bool) error {
	vm, err := m.Get(name)
	if err != nil {
		return err
	}

	if vm.State != "running" {
		return fmt.Errorf("VM is not running")
	}

	pidFile := filepath.Join(m.vmDir, vm.Name, "qemu.pid")
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return fmt.Errorf("invalid PID: %w", err)
	}

	signal := "-TERM"
	if force {
		signal = "-KILL"
	}

	if err := exec.Command("kill", signal, strconv.Itoa(pid)).Run(); err != nil {
		return fmt.Errorf("failed to stop VM: %w", err)
	}

	vm.State = "stopped"
	return m.saveVM(vm)
}

// Destroy destroys a VM
func (m *Manager) Destroy(name string) error {
	vm, err := m.Get(name)
	if err != nil {
		return err
	}

	// Stop if running
	if vm.State == "running" {
		if err := m.Stop(name, true); err != nil {
			return err
		}
	}

	// Remove VM directory
	vmPath := filepath.Join(m.vmDir, name)
	return os.RemoveAll(vmPath)
}

// Get gets a VM by name
func (m *Manager) Get(name string) (*VM, error) {
	configPath := filepath.Join(m.vmDir, name, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("VM not found: %s", name)
	}

	var vm VM
	if err := json.Unmarshal(data, &vm); err != nil {
		return nil, fmt.Errorf("failed to parse VM config: %w", err)
	}

	// Check actual state
	pidFile := filepath.Join(m.vmDir, name, "qemu.pid")
	if _, err := os.Stat(pidFile); err == nil {
		pidData, _ := os.ReadFile(pidFile)
		pid, _ := strconv.Atoi(strings.TrimSpace(string(pidData)))
		if pid > 0 {
			if err := exec.Command("kill", "-0", strconv.Itoa(pid)).Run(); err == nil {
				vm.State = "running"
			} else {
				vm.State = "stopped"
				os.Remove(pidFile)
			}
		}
	}

	return &vm, nil
}

// List lists all VMs
func (m *Manager) List() ([]*VM, error) {
	entries, err := os.ReadDir(m.vmDir)
	if err != nil {
		return nil, err
	}

	var vms []*VM
	for _, entry := range entries {
		if entry.IsDir() {
			vm, err := m.Get(entry.Name())
			if err == nil {
				vms = append(vms, vm)
			}
		}
	}

	return vms, nil
}

// SSH connects to a VM via SSH
func (m *Manager) SSH(name string, user string) error {
	vm, err := m.Get(name)
	if err != nil {
		return err
	}

	if vm.State != "running" {
		return fmt.Errorf("VM is not running")
	}

	if vm.SSH == 0 {
		return fmt.Errorf("SSH port not configured")
	}

	if user == "" {
		user = "root"
	}

	cmd := exec.Command("ssh",
		"-p", strconv.Itoa(vm.SSH),
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		fmt.Sprintf("%s@localhost", user),
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Console connects to VM console
func (m *Manager) Console(name string) error {
	vm, err := m.Get(name)
	if err != nil {
		return err
	}

	if vm.State != "running" {
		return fmt.Errorf("VM is not running")
	}

	monitorSocket := filepath.Join(m.vmDir, vm.Name, "monitor.sock")
	cmd := exec.Command("socat", "-", fmt.Sprintf("UNIX-CONNECT:%s", monitorSocket))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Snapshot creates a VM snapshot
func (m *Manager) Snapshot(name string, snapshotName string) error {
	vm, err := m.Get(name)
	if err != nil {
		return err
	}

	snapshotPath := filepath.Join(m.vmDir, name, "snapshots", snapshotName+".qcow2")
	os.MkdirAll(filepath.Dir(snapshotPath), 0755)

	return exec.Command("qemu-img", "snapshot", "-c", snapshotName, vm.DiskPath).Run()
}

// RestoreSnapshot restores a VM snapshot
func (m *Manager) RestoreSnapshot(name string, snapshotName string) error {
	vm, err := m.Get(name)
	if err != nil {
		return err
	}

	if vm.State == "running" {
		return fmt.Errorf("cannot restore snapshot while VM is running")
	}

	return exec.Command("qemu-img", "snapshot", "-a", snapshotName, vm.DiskPath).Run()
}

// ListSnapshots lists VM snapshots
func (m *Manager) ListSnapshots(name string) ([]string, error) {
	vm, err := m.Get(name)
	if err != nil {
		return nil, err
	}

	out, err := exec.Command("qemu-img", "snapshot", "-l", vm.DiskPath).Output()
	if err != nil {
		return nil, err
	}

	var snapshots []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "ID") || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			snapshots = append(snapshots, fields[1])
		}
	}

	return snapshots, nil
}

func (m *Manager) createDisk(path string, sizeGB int) error {
	return exec.Command("qemu-img", "create", "-f", "qcow2", path, fmt.Sprintf("%dG", sizeGB)).Run()
}

func (m *Manager) saveVM(vm *VM) error {
	configPath := filepath.Join(m.vmDir, vm.Name, "config.json")
	data, err := json.MarshalIndent(vm, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func findAvailablePort(start int) int {
	// Simple implementation - in production would check if port is actually available
	return start
}
