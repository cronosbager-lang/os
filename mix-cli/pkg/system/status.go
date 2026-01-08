package system

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ServiceStatus represents a systemd service status
type ServiceStatus struct {
	Name        string
	Description string
	LoadState   string
	ActiveState string
	SubState    string
	MainPID     int
	Memory      string
	CPU         string
}

// ProcessInfo represents process information
type ProcessInfo struct {
	PID     int
	User    string
	CPU     float64
	Memory  float64
	VSZ     uint64
	RSS     uint64
	TTY     string
	State   string
	Start   string
	Time    string
	Command string
}

// LoadAverage represents system load
type LoadAverage struct {
	Load1  float64
	Load5  float64
	Load15 float64
}

// GetServiceStatus gets the status of a systemd service
func GetServiceStatus(name string) (*ServiceStatus, error) {
	out, err := exec.Command("systemctl", "show", name,
		"--property=Description,LoadState,ActiveState,SubState,MainPID,MemoryCurrent,CPUUsageNSec").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get service status: %w", err)
	}

	status := &ServiceStatus{Name: name}

	for _, line := range strings.Split(string(out), "\n") {
		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			switch key {
			case "Description":
				status.Description = value
			case "LoadState":
				status.LoadState = value
			case "ActiveState":
				status.ActiveState = value
			case "SubState":
				status.SubState = value
			case "MainPID":
				status.MainPID, _ = strconv.Atoi(value)
			case "MemoryCurrent":
				if bytes, err := strconv.ParseUint(value, 10, 64); err == nil {
					status.Memory = FormatBytes(bytes)
				}
			case "CPUUsageNSec":
				if ns, err := strconv.ParseUint(value, 10, 64); err == nil {
					status.CPU = fmt.Sprintf("%.2fs", float64(ns)/1e9)
				}
			}
		}
	}

	return status, nil
}

// ListServices lists all systemd services
func ListServices() ([]ServiceStatus, error) {
	out, err := exec.Command("systemctl", "list-units", "--type=service", "--all", "--no-pager", "--plain").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	var services []ServiceStatus
	scanner := bufio.NewScanner(strings.NewReader(string(out)))

	// Skip header
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}

		fields := strings.Fields(line)
		if len(fields) >= 4 {
			services = append(services, ServiceStatus{
				Name:        strings.TrimSuffix(fields[0], ".service"),
				LoadState:   fields[1],
				ActiveState: fields[2],
				SubState:    fields[3],
				Description: strings.Join(fields[4:], " "),
			})
		}
	}

	return services, nil
}

// StartService starts a systemd service
func StartService(name string) error {
	return exec.Command("systemctl", "start", name).Run()
}

// StopService stops a systemd service
func StopService(name string) error {
	return exec.Command("systemctl", "stop", name).Run()
}

// RestartService restarts a systemd service
func RestartService(name string) error {
	return exec.Command("systemctl", "restart", name).Run()
}

// EnableService enables a systemd service
func EnableService(name string) error {
	return exec.Command("systemctl", "enable", name).Run()
}

// DisableService disables a systemd service
func DisableService(name string) error {
	return exec.Command("systemctl", "disable", name).Run()
}

// GetLoadAverage returns system load average
func GetLoadAverage() (*LoadAverage, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return nil, fmt.Errorf("unexpected loadavg format")
	}

	load := &LoadAverage{}
	load.Load1, _ = strconv.ParseFloat(fields[0], 64)
	load.Load5, _ = strconv.ParseFloat(fields[1], 64)
	load.Load15, _ = strconv.ParseFloat(fields[2], 64)

	return load, nil
}

// GetProcessList returns list of running processes
func GetProcessList() ([]ProcessInfo, error) {
	out, err := exec.Command("ps", "aux", "--no-headers").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get process list: %w", err)
	}

	var processes []ProcessInfo
	scanner := bufio.NewScanner(strings.NewReader(string(out)))

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 11 {
			continue
		}

		proc := ProcessInfo{
			User:    fields[0],
			TTY:     fields[6],
			State:   fields[7],
			Start:   fields[8],
			Time:    fields[9],
			Command: strings.Join(fields[10:], " "),
		}

		proc.PID, _ = strconv.Atoi(fields[1])
		proc.CPU, _ = strconv.ParseFloat(fields[2], 64)
		proc.Memory, _ = strconv.ParseFloat(fields[3], 64)
		proc.VSZ, _ = strconv.ParseUint(fields[4], 10, 64)
		proc.RSS, _ = strconv.ParseUint(fields[5], 10, 64)

		processes = append(processes, proc)
	}

	return processes, nil
}

// GetTopProcesses returns top N processes by CPU or memory
func GetTopProcesses(n int, sortBy string) ([]ProcessInfo, error) {
	processes, err := GetProcessList()
	if err != nil {
		return nil, err
	}

	// Sort
	switch sortBy {
	case "cpu":
		for i := 0; i < len(processes)-1; i++ {
			for j := i + 1; j < len(processes); j++ {
				if processes[j].CPU > processes[i].CPU {
					processes[i], processes[j] = processes[j], processes[i]
				}
			}
		}
	case "memory", "mem":
		for i := 0; i < len(processes)-1; i++ {
			for j := i + 1; j < len(processes); j++ {
				if processes[j].Memory > processes[i].Memory {
					processes[i], processes[j] = processes[j], processes[i]
				}
			}
		}
	}

	if n > len(processes) {
		n = len(processes)
	}

	return processes[:n], nil
}

// KillProcess kills a process by PID
func KillProcess(pid int, signal string) error {
	if signal == "" {
		signal = "TERM"
	}
	return exec.Command("kill", "-"+signal, strconv.Itoa(pid)).Run()
}

// GetBootTime returns system boot time
func GetBootTime() (time.Time, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Time{}, err
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "btime ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				ts, _ := strconv.ParseInt(fields[1], 10, 64)
				return time.Unix(ts, 0), nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("boot time not found")
}

// IsServiceRunning checks if a service is running
func IsServiceRunning(name string) bool {
	status, err := GetServiceStatus(name)
	if err != nil {
		return false
	}
	return status.ActiveState == "active"
}

// WaitForService waits for a service to reach a state
func WaitForService(name string, state string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		status, err := GetServiceStatus(name)
		if err == nil && status.ActiveState == state {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for service %s to reach state %s", name, state)
}
