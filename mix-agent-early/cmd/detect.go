package cmd

import (
	"fmt"

	"github.com/mixos/mix-agent-early/pkg/hardware"
)

func RunDetect() {
	fmt.Println("========================================")
	fmt.Println("MIXOS Hardware Detection")
	fmt.Println("========================================")

	hw := hardware.Detect()

	// CPU
	fmt.Println("\nCPU:")
	fmt.Printf("  Model:  %s\n", hw.CPU.Model)
	fmt.Printf("  Vendor: %s\n", hw.CPU.Vendor)
	fmt.Printf("  Cores:  %d\n", hw.CPU.Cores)

	// Memory
	fmt.Println("\nMemory:")
	fmt.Printf("  Total:     %d MB\n", hw.Memory.Total/1024/1024)
	fmt.Printf("  Available: %d MB\n", hw.Memory.Available/1024/1024)

	// Disks
	fmt.Println("\nDisks:")
	for _, disk := range hw.Disks {
		fmt.Printf("  %s: %s (%d GB)\n", disk.Device, disk.Model, disk.Size/1024/1024/1024)
	}

	// Network
	fmt.Println("\nNetwork Interfaces:")
	for _, iface := range hw.Network {
		fmt.Printf("  %s: %s\n", iface.Name, iface.MAC)
	}

	// Required modules
	fmt.Println("\nRequired Kernel Modules:")
	modules := hardware.GetRequiredModules(hw)
	for _, mod := range modules {
		fmt.Printf("  - %s\n", mod)
	}
}
