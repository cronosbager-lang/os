package hardware

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type HardwareInfo struct {
	CPU     CPUInfo
	Memory  MemoryInfo
	Disks   []DiskInfo
	Network []NetworkInfo
}

type CPUInfo struct {
	Model  string
	Vendor string
	Cores  int
}

type MemoryInfo struct {
	Total     uint64
	Available uint64
}

type DiskInfo struct {
	Device string
	Model  string
	Size   uint64
	Type   string // "sata", "nvme", "usb"
}

type NetworkInfo struct {
	Name string
	MAC  string
}

func Detect() *HardwareInfo {
	hw := &HardwareInfo{}
	hw.CPU = detectCPU()
	hw.Memory = detectMemory()
	hw.Disks = detectDisks()
	hw.Network = detectNetwork()
	return hw
}

func detectCPU() CPUInfo {
	info := CPUInfo{
		Model:  "Unknown",
		Vendor: "Unknown",
		Cores:  1,
	}

	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return info
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.Model = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "vendor_id") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.Vendor = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "processor") {
			info.Cores++
		}
	}

	// Adjust cores count (starts from 0)
	if info.Cores > 0 {
		info.Cores--
	}
	info.Cores++

	return info
}

func detectMemory() MemoryInfo {
	info := MemoryInfo{}

	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return info
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			info.Total = parseMemValue(line)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			info.Available = parseMemValue(line)
		}
	}

	return info
}

func parseMemValue(line string) uint64 {
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		val, _ := strconv.ParseUint(parts[1], 10, 64)
		return val * 1024 // Convert from kB to bytes
	}
	return 0
}

func detectDisks() []DiskInfo {
	var disks []DiskInfo

	// Check /sys/block for block devices
	entries, err := os.ReadDir("/sys/block")
	if err != nil {
		return disks
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip loop, ram, and dm devices
		if strings.HasPrefix(name, "loop") ||
			strings.HasPrefix(name, "ram") ||
			strings.HasPrefix(name, "dm-") {
			continue
		}

		disk := DiskInfo{
			Device: "/dev/" + name,
		}

		// Get size
		sizePath := filepath.Join("/sys/block", name, "size")
		if data, err := os.ReadFile(sizePath); err == nil {
			sectors, _ := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
			disk.Size = sectors * 512 // 512 bytes per sector
		}

		// Get model
		modelPath := filepath.Join("/sys/block", name, "device/model")
		if data, err := os.ReadFile(modelPath); err == nil {
			disk.Model = strings.TrimSpace(string(data))
		}

		// Determine type
		if strings.HasPrefix(name, "nvme") {
			disk.Type = "nvme"
		} else if strings.HasPrefix(name, "sd") {
			disk.Type = "sata"
		} else if strings.HasPrefix(name, "vd") {
			disk.Type = "virtio"
		}

		disks = append(disks, disk)
	}

	return disks
}

func detectNetwork() []NetworkInfo {
	var interfaces []NetworkInfo

	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return interfaces
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == "lo" {
			continue
		}

		iface := NetworkInfo{
			Name: name,
		}

		// Get MAC address
		macPath := filepath.Join("/sys/class/net", name, "address")
		if data, err := os.ReadFile(macPath); err == nil {
			iface.MAC = strings.TrimSpace(string(data))
		}

		interfaces = append(interfaces, iface)
	}

	return interfaces
}

func GetRequiredModules(hw *HardwareInfo) []string {
	modules := []string{}

	// Always needed
	modules = append(modules, "ext4", "squashfs", "overlay")

	// Disk modules based on detected hardware
	for _, disk := range hw.Disks {
		switch disk.Type {
		case "nvme":
			modules = append(modules, "nvme", "nvme_core")
		case "sata":
			modules = append(modules, "ahci", "sd_mod")
		case "virtio":
			modules = append(modules, "virtio_blk", "virtio_pci")
		}
	}

	// USB storage (always useful)
	modules = append(modules, "usb_storage", "uas")

	// Network modules
	if len(hw.Network) > 0 {
		modules = append(modules, "e1000", "e1000e", "r8169", "virtio_net")
	}

	// Remove duplicates
	seen := make(map[string]bool)
	unique := []string{}
	for _, mod := range modules {
		if !seen[mod] {
			seen[mod] = true
			unique = append(unique, mod)
		}
	}

	return unique
}
