package cmd

import (
	"fmt"
	"os"

	"github.com/mixos/mix-agent-early/pkg/boot"
	"github.com/mixos/mix-agent-early/pkg/hardware"
	"github.com/mixos/mix-agent-early/pkg/logger"
)

func RunInit() {
	log := logger.New("mix-agent-early")
	log.Info("MIXOS AI Agent Early Boot initialized")

	// Step 1: Detect hardware
	log.Info("Detecting hardware...")
	hw := hardware.Detect()
	log.Info("CPU: %s (%d cores)", hw.CPU.Model, hw.CPU.Cores)
	log.Info("RAM: %d MB", hw.Memory.Total/1024/1024)
	log.Info("Disks: %d found", len(hw.Disks))

	// Step 2: Determine required modules
	log.Info("Analyzing required kernel modules...")
	modules := hardware.GetRequiredModules(hw)
	log.Info("Required modules: %v", modules)

	// Step 3: Load modules
	log.Info("Loading kernel modules...")
	for _, mod := range modules {
		if err := boot.LoadModule(mod); err != nil {
			log.Warn("Failed to load module %s: %v", mod, err)
		} else {
			log.Info("Loaded: %s", mod)
		}
	}

	// Step 4: Find root filesystem
	log.Info("Searching for root filesystem...")
	rootDev, err := boot.FindRootFS()
	if err != nil {
		log.Error("Failed to find root filesystem: %v", err)
		RunEmergency()
		return
	}
	log.Info("Found root filesystem: %s", rootDev)

	// Step 5: Mount root filesystem
	log.Info("Mounting root filesystem...")
	if err := boot.MountRoot(rootDev, "/newroot"); err != nil {
		log.Error("Failed to mount root: %v", err)
		RunEmergency()
		return
	}

	// Step 6: Validate root filesystem
	log.Info("Validating root filesystem...")
	if err := boot.ValidateRoot("/newroot"); err != nil {
		log.Error("Root filesystem validation failed: %v", err)
		RunEmergency()
		return
	}

	log.Info("Root filesystem ready, preparing to switch root...")

	// The actual switch_root is done by the init script
	// We just prepare everything and exit successfully
	os.Exit(0)
}

func RunEmergency() {
	fmt.Println("\n========================================")
	fmt.Println("MIXOS Emergency Shell")
	fmt.Println("========================================")
	fmt.Println("The system failed to boot normally.")
	fmt.Println("You have been dropped to an emergency shell.")
	fmt.Println("")
	fmt.Println("Available commands:")
	fmt.Println("  mix-agent-early detect   - Detect hardware")
	fmt.Println("  mix-agent-early analyze  - Analyze boot issues")
	fmt.Println("  lsblk                    - List block devices")
	fmt.Println("  mount                    - Mount filesystems")
	fmt.Println("  exit                     - Reboot")
	fmt.Println("========================================")

	// Start emergency shell
	os.Exit(1)
}
