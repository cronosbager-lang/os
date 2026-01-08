package system

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// SystemInfo contains system information
type SystemInfo struct {
	Hostname     string
	OS           string
	Kernel       string
	Architecture string
	CPU          CPUInfo
	Memory       MemoryInfo
	Disk         []DiskInfo
	Network      []NetworkInfo
	Uptime       string
}

// CPUInfo contains CPU information
type CPUInfo struct {
	Model   string
	Vendor  string
	Cores   int
	Threads int
	MHz     float64
}

// MemoryInfo contains memory information
type MemoryInfo struct {
	Total     uint64
	Used      uint64
	Free      uint64
	Available uint64
	SwapTotal uint64
	SwapUsed  uint64
}

// DiskInfo contains disk information
type DiskInfo struct {
	Device     string
	MountPoint string
	FSType     string
	Total      uint64
	Used       uint64
	Free       uint64
	UsedPct    float64
}

// NetworkInfo contains network interface information
type NetworkInfo struct {
	Name      string
	IPv4      string
	IPv6      string
	MAC       string
	State     string
	RxBytes   uint64
	TxBytes   uint64
}

// GetSystemInfo gathers all system information
func GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{
		Architecture: runtime.GOARCH,
	}

	// Hostname
	hostname, _ := os.Hostname()
	info.Hostname = hostname

	// OS info
	info.OS = getOSInfo()
	info.Kernel = getKernelVersion()

	// CPU info
	info.CPU = getCPUInfo()

	// Memory info
	info.Memory = getMemoryInfo()

	// Disk info
	info.Disk = getDiskInfo()

	// Network info
	info.Network = getNetworkInfo()

	// Uptime
	info.Uptime = getUptime()

	return info, nil
}

func getOSInfo() string {
	// Try /etc/os-release first
	file, err := os.Open("/etc/os-release")
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				name := strings.TrimPrefix(line, "PRETTY_NAME=")
				return strings.Trim(name, "\"")
			}
		}
	}

	return runtime.GOOS
}

func getKernelVersion() string {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return ""
	}

	parts := strings.Fields(string(data))
	if len(parts) >= 3 {
		return parts[2]
	}
	return string(data)
}

func getCPUInfo() CPUInfo {
	info := CPUInfo{
		Model:  "Unknown",
		Vendor: "Unknown",
		Cores:  runtime.NumCPU(),
	}

	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return info
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	threads := 0

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
		} else if strings.HasPrefix(line, "cpu MHz") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.MHz, _ = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			}
		} else if strings.HasPrefix(line, "processor") {
			threads++
		} else if strings.HasPrefix(line, "cpu cores") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.Cores, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
			}
		}
	}

	info.Threads = threads
	return info
}

func getMemoryInfo() MemoryInfo {
	info := MemoryInfo{}

	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return info
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		value, _ := strconv.ParseUint(parts[1], 10, 64)
		value *= 1024 // Convert from kB to bytes

		switch parts[0] {
		case "MemTotal:":
			info.Total = value
		case "MemFree:":
			info.Free = value
		case "MemAvailable:":
			info.Available = value
		case "SwapTotal:":
			info.SwapTotal = value
		case "SwapFree:":
			info.SwapUsed = info.SwapTotal - value
		}
	}

	info.Used = info.Total - info.Available
	return info
}

func getDiskInfo() []DiskInfo {
	var disks []DiskInfo

	file, err := os.Open("/proc/mounts")
	if err != nil {
		return disks
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}

		device := fields[0]
		mountPoint := fields[1]
		fsType := fields[2]

		// Skip virtual filesystems
		if strings.HasPrefix(device, "/dev/") {
			disk := DiskInfo{
				Device:     device,
				MountPoint: mountPoint,
				FSType:     fsType,
			}

			// Get disk usage
			var stat syscallStatfs
			if err := statfs(mountPoint, &stat); err == nil {
				disk.Total = stat.Blocks * uint64(stat.Bsize)
				disk.Free = stat.Bavail * uint64(stat.Bsize)
				disk.Used = disk.Total - disk.Free
				if disk.Total > 0 {
					disk.UsedPct = float64(disk.Used) / float64(disk.Total) * 100
				}
			}

			disks = append(disks, disk)
		}
	}

	return disks
}

func getNetworkInfo() []NetworkInfo {
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

		// Get state
		statePath := filepath.Join("/sys/class/net", name, "operstate")
		if data, err := os.ReadFile(statePath); err == nil {
			iface.State = strings.TrimSpace(string(data))
		}

		// Get RX/TX bytes
		rxPath := filepath.Join("/sys/class/net", name, "statistics/rx_bytes")
		if data, err := os.ReadFile(rxPath); err == nil {
			iface.RxBytes, _ = strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
		}

		txPath := filepath.Join("/sys/class/net", name, "statistics/tx_bytes")
		if data, err := os.ReadFile(txPath); err == nil {
			iface.TxBytes, _ = strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
		}

		// Get IP addresses using ip command
		if out, err := exec.Command("ip", "-4", "addr", "show", name).Output(); err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, "inet ") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						iface.IPv4 = strings.Split(fields[1], "/")[0]
					}
				}
			}
		}

		interfaces = append(interfaces, iface)
	}

	return interfaces
}

func getUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return ""
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return ""
	}

	seconds, _ := strconv.ParseFloat(fields[0], 64)
	return formatDuration(int64(seconds))
}

func formatDuration(seconds int64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// syscallStatfs is a simplified statfs structure
type syscallStatfs struct {
	Bsize  int64
	Blocks uint64
	Bfree  uint64
	Bavail uint64
}

// statfs is a wrapper for the statfs syscall
func statfs(path string, stat *syscallStatfs) error {
	// Use df command as fallback
	out, err := exec.Command("df", "-B1", path).Output()
	if err != nil {
		return err
	}

	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return fmt.Errorf("unexpected df output")
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return fmt.Errorf("unexpected df output format")
	}

	stat.Bsize = 1
	stat.Blocks, _ = strconv.ParseUint(fields[1], 10, 64)
	stat.Bavail, _ = strconv.ParseUint(fields[3], 10, 64)
	stat.Bfree = stat.Bavail

	return nil
}

// FormatBytes formats bytes to human-readable string
func FormatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
