package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show system status",
	Long:  `Display current system status including OS info, AI agent status, and resource usage.`,
	Run:   runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Println(cyan("═══════════════════════════════════════"))
	fmt.Println(cyan("         MIXOS GO System Status        "))
	fmt.Println(cyan("═══════════════════════════════════════"))
	fmt.Println()

	// OS Info
	fmt.Println(cyan("System Information"))
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("  OS:           %s\n", getOSInfo())
	fmt.Printf("  Kernel:       %s\n", getKernelVersion())
	fmt.Printf("  Architecture: %s\n", runtime.GOARCH)
	fmt.Printf("  Hostname:     %s\n", getHostname())
	fmt.Println()

	// AI Agent Status
	fmt.Println(cyan("AI Agent"))
	fmt.Println(strings.Repeat("─", 40))
	agentStatus := getAgentStatus()
	if agentStatus == "running" {
		fmt.Printf("  Status:       %s\n", green("● Running"))
	} else {
		fmt.Printf("  Status:       %s\n", yellow("○ Stopped"))
	}
	fmt.Printf("  Model:        mix-small-1.1b\n")
	fmt.Println()

	// Docker Status
	fmt.Println(cyan("Docker"))
	fmt.Println(strings.Repeat("─", 40))
	dockerStatus := getDockerStatus()
	if dockerStatus == "running" {
		fmt.Printf("  Status:       %s\n", green("● Running"))
		fmt.Printf("  Containers:   %s\n", getContainerCount())
	} else {
		fmt.Printf("  Status:       %s\n", yellow("○ Stopped"))
	}
	fmt.Println()

	// Resource Usage
	fmt.Println(cyan("Resources"))
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("  CPU Cores:    %d\n", runtime.NumCPU())
	fmt.Printf("  Memory:       %s\n", getMemoryInfo())
	fmt.Printf("  Disk:         %s\n", getDiskInfo())
	fmt.Println()
}

func getOSInfo() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "MIXOS GO"
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
		}
	}
	return "MIXOS GO"
}

func getKernelVersion() string {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func getAgentStatus() string {
	out, err := exec.Command("systemctl", "is-active", "mixos-agent").Output()
	if err != nil {
		return "stopped"
	}
	status := strings.TrimSpace(string(out))
	if status == "active" {
		return "running"
	}
	return "stopped"
}

func getDockerStatus() string {
	out, err := exec.Command("systemctl", "is-active", "docker").Output()
	if err != nil {
		return "stopped"
	}
	status := strings.TrimSpace(string(out))
	if status == "active" {
		return "running"
	}
	return "stopped"
}

func getContainerCount() string {
	out, err := exec.Command("docker", "ps", "-q").Output()
	if err != nil {
		return "N/A"
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return "0 running"
	}
	return fmt.Sprintf("%d running", len(lines))
}

func getMemoryInfo() string {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return "unknown"
	}
	lines := strings.Split(string(data), "\n")
	var total, available int64
	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal:") {
			fmt.Sscanf(line, "MemTotal: %d kB", &total)
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			fmt.Sscanf(line, "MemAvailable: %d kB", &available)
		}
	}
	if total > 0 {
		used := total - available
		return fmt.Sprintf("%.1f GB / %.1f GB (%.0f%%)",
			float64(used)/1024/1024,
			float64(total)/1024/1024,
			float64(used)/float64(total)*100)
	}
	return "unknown"
}

func getDiskInfo() string {
	out, err := exec.Command("df", "-h", "/").Output()
	if err != nil {
		return "unknown"
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) >= 2 {
		fields := strings.Fields(lines[1])
		if len(fields) >= 5 {
			return fmt.Sprintf("%s / %s (%s used)", fields[2], fields[1], fields[4])
		}
	}
	return "unknown"
}
