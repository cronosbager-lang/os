package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/mixos/mix-agent-early/pkg/boot"
	"github.com/mixos/mix-agent-early/pkg/hardware"
)

func RunAnalyze() {
	fmt.Println("========================================")
	fmt.Println("MIXOS Boot Analysis")
	fmt.Println("========================================")

	issues := []string{}
	suggestions := []string{}

	// Check hardware
	fmt.Println("\n[1/4] Checking hardware...")
	hw := hardware.Detect()
	if hw.Memory.Total < 512*1024*1024 {
		issues = append(issues, "Low memory (<512MB)")
		suggestions = append(suggestions, "MIXOS requires at least 512MB RAM")
	}
	if len(hw.Disks) == 0 {
		issues = append(issues, "No disks detected")
		suggestions = append(suggestions, "Check disk connections and load storage drivers")
	}
	fmt.Println("  Hardware check complete")

	// Check kernel modules
	fmt.Println("\n[2/4] Checking kernel modules...")
	modules := hardware.GetRequiredModules(hw)
	for _, mod := range modules {
		if !isModuleLoaded(mod) {
			issues = append(issues, fmt.Sprintf("Module not loaded: %s", mod))
			suggestions = append(suggestions, fmt.Sprintf("Try: modprobe %s", mod))
		}
	}
	fmt.Println("  Module check complete")

	// Check for root filesystem
	fmt.Println("\n[3/4] Searching for root filesystem...")
	rootDev, err := boot.FindRootFS()
	if err != nil {
		issues = append(issues, "Root filesystem not found")
		suggestions = append(suggestions, "Check boot parameters (root=)")
		suggestions = append(suggestions, "Verify disk partitions with: lsblk")
	} else {
		fmt.Printf("  Found: %s\n", rootDev)
	}

	// Check mount points
	fmt.Println("\n[4/4] Checking mount points...")
	if _, err := os.Stat("/newroot"); os.IsNotExist(err) {
		issues = append(issues, "/newroot mount point missing")
		suggestions = append(suggestions, "Create mount point: mkdir /newroot")
	}
	fmt.Println("  Mount point check complete")

	// Report
	fmt.Println("\n========================================")
	fmt.Println("Analysis Results")
	fmt.Println("========================================")

	if len(issues) == 0 {
		fmt.Println("\n✓ No issues found")
		fmt.Println("  The system should be able to boot normally.")
	} else {
		fmt.Printf("\n✗ Found %d issue(s):\n", len(issues))
		for i, issue := range issues {
			fmt.Printf("  %d. %s\n", i+1, issue)
		}

		fmt.Println("\nSuggestions:")
		for _, suggestion := range suggestions {
			fmt.Printf("  → %s\n", suggestion)
		}
	}
}

func isModuleLoaded(name string) bool {
	data, err := os.ReadFile("/proc/modules")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), name)
}
