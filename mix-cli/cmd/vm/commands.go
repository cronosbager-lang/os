package vm

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	vmMemory string
	vmCPUs   int
	vmDisk   string
)

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new VM",
	Args:  cobra.ExactArgs(1),
	Run:   runCreate,
}

var startCmd = &cobra.Command{
	Use:   "start [name]",
	Short: "Start a VM",
	Args:  cobra.ExactArgs(1),
	Run:   runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop a VM",
	Args:  cobra.ExactArgs(1),
	Run:   runStop,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all VMs",
	Run:   runList,
}

var destroyCmd = &cobra.Command{
	Use:   "destroy [name]",
	Short: "Destroy a VM",
	Args:  cobra.ExactArgs(1),
	Run:   runDestroy,
}

func init() {
	createCmd.Flags().StringVarP(&vmMemory, "memory", "m", "2G", "memory size")
	createCmd.Flags().IntVarP(&vmCPUs, "cpus", "c", 2, "number of CPUs")
	createCmd.Flags().StringVarP(&vmDisk, "disk", "d", "20G", "disk size")
}

func runCreate(cmd *cobra.Command, args []string) {
	name := args[0]
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("%s Creating VM: %s\n", cyan("→"), name)
	fmt.Printf("  Memory: %s\n", vmMemory)
	fmt.Printf("  CPUs:   %d\n", vmCPUs)
	fmt.Printf("  Disk:   %s\n", vmDisk)

	// Create VM directory
	vmDir := fmt.Sprintf("/var/lib/mixos/vms/%s", name)
	if err := os.MkdirAll(vmDir, 0755); err != nil {
		fmt.Printf("Failed to create VM directory: %v\n", err)
		os.Exit(1)
	}

	// Create disk image
	diskPath := fmt.Sprintf("%s/disk.qcow2", vmDir)
	qemuImg := exec.Command("qemu-img", "create", "-f", "qcow2", diskPath, vmDisk)
	if err := qemuImg.Run(); err != nil {
		fmt.Printf("Failed to create disk image: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s VM '%s' created\n", green("✓"), name)
}

func runStart(cmd *cobra.Command, args []string) {
	name := args[0]
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("%s Starting VM: %s\n", cyan("→"), name)

	vmDir := fmt.Sprintf("/var/lib/mixos/vms/%s", name)
	diskPath := fmt.Sprintf("%s/disk.qcow2", vmDir)

	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		fmt.Printf("VM '%s' not found\n", name)
		os.Exit(1)
	}

	// Start QEMU in background
	qemu := exec.Command("qemu-system-x86_64",
		"-enable-kvm",
		"-m", "2G",
		"-smp", "2",
		"-drive", fmt.Sprintf("file=%s,format=qcow2", diskPath),
		"-daemonize",
		"-pidfile", fmt.Sprintf("%s/qemu.pid", vmDir),
	)
	if err := qemu.Run(); err != nil {
		fmt.Printf("Failed to start VM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s VM '%s' started\n", green("✓"), name)
}

func runStop(cmd *cobra.Command, args []string) {
	name := args[0]
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("%s Stopping VM: %s\n", cyan("→"), name)

	vmDir := fmt.Sprintf("/var/lib/mixos/vms/%s", name)
	pidFile := fmt.Sprintf("%s/qemu.pid", vmDir)

	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		fmt.Printf("VM '%s' is not running\n", name)
		os.Exit(1)
	}

	pid := strings.TrimSpace(string(pidData))
	kill := exec.Command("kill", pid)
	if err := kill.Run(); err != nil {
		fmt.Printf("Failed to stop VM: %v\n", err)
		os.Exit(1)
	}

	os.Remove(pidFile)
	fmt.Printf("%s VM '%s' stopped\n", green("✓"), name)
}

func runList(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	vmBase := "/var/lib/mixos/vms"
	entries, err := os.ReadDir(vmBase)
	if err != nil {
		fmt.Println("No VMs found")
		return
	}

	fmt.Println(cyan("Virtual Machines"))
	fmt.Println(strings.Repeat("─", 40))

	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			pidFile := fmt.Sprintf("%s/%s/qemu.pid", vmBase, name)
			if _, err := os.Stat(pidFile); err == nil {
				fmt.Printf("  %s %s\n", green("●"), name)
			} else {
				fmt.Printf("  %s %s\n", yellow("○"), name)
			}
		}
	}
}

func runDestroy(cmd *cobra.Command, args []string) {
	name := args[0]
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("%s Destroying VM: %s\n", cyan("→"), name)

	vmDir := fmt.Sprintf("/var/lib/mixos/vms/%s", name)

	// Stop if running
	pidFile := fmt.Sprintf("%s/qemu.pid", vmDir)
	if pidData, err := os.ReadFile(pidFile); err == nil {
		pid := strings.TrimSpace(string(pidData))
		exec.Command("kill", pid).Run()
	}

	// Remove VM directory
	if err := os.RemoveAll(vmDir); err != nil {
		fmt.Printf("Failed to destroy VM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s VM '%s' destroyed\n", green("✓"), name)
}
