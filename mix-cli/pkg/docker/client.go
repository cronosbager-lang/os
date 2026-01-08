package docker

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Client for Docker operations
type Client struct {
	socketPath string
}

// NewClient creates a new Docker client
func NewClient() *Client {
	return &Client{
		socketPath: "/var/run/docker.sock",
	}
}

// Container represents a Docker container
type Container struct {
	ID      string
	Name    string
	Image   string
	Command string
	Created string
	Status  string
	Ports   string
	State   string
}

// Image represents a Docker image
type Image struct {
	ID         string
	Repository string
	Tag        string
	Created    string
	Size       string
}

// Network represents a Docker network
type Network struct {
	ID     string
	Name   string
	Driver string
	Scope  string
}

// Volume represents a Docker volume
type Volume struct {
	Name       string
	Driver     string
	Mountpoint string
}

// ContainerStats represents container statistics
type ContainerStats struct {
	CPUPercent    float64
	MemoryUsage   uint64
	MemoryLimit   uint64
	MemoryPercent float64
	NetworkRx     uint64
	NetworkTx     uint64
	BlockRead     uint64
	BlockWrite    uint64
}

// IsAvailable checks if Docker is available
func (c *Client) IsAvailable() bool {
	err := exec.Command("docker", "info").Run()
	return err == nil
}

// ListContainers lists all containers
func (c *Client) ListContainers(all bool) ([]Container, error) {
	args := []string{"ps", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Command}}\t{{.CreatedAt}}\t{{.Status}}\t{{.Ports}}\t{{.State}}"}
	if all {
		args = append(args, "-a")
	}

	out, err := exec.Command("docker", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var containers []Container
	scanner := bufio.NewScanner(strings.NewReader(string(out)))

	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) >= 8 {
			containers = append(containers, Container{
				ID:      fields[0],
				Name:    fields[1],
				Image:   fields[2],
				Command: fields[3],
				Created: fields[4],
				Status:  fields[5],
				Ports:   fields[6],
				State:   fields[7],
			})
		}
	}

	return containers, nil
}

// GetContainer gets a specific container
func (c *Client) GetContainer(id string) (*Container, error) {
	out, err := exec.Command("docker", "inspect", "--format",
		"{{.Id}}\t{{.Name}}\t{{.Config.Image}}\t{{.Created}}\t{{.State.Status}}",
		id).Output()
	if err != nil {
		return nil, fmt.Errorf("container not found: %w", err)
	}

	fields := strings.Split(strings.TrimSpace(string(out)), "\t")
	if len(fields) < 5 {
		return nil, fmt.Errorf("unexpected output format")
	}

	return &Container{
		ID:      fields[0][:12],
		Name:    strings.TrimPrefix(fields[1], "/"),
		Image:   fields[2],
		Created: fields[3],
		State:   fields[4],
	}, nil
}

// StartContainer starts a container
func (c *Client) StartContainer(id string) error {
	return exec.Command("docker", "start", id).Run()
}

// StopContainer stops a container
func (c *Client) StopContainer(id string, timeout int) error {
	return exec.Command("docker", "stop", "-t", fmt.Sprintf("%d", timeout), id).Run()
}

// RestartContainer restarts a container
func (c *Client) RestartContainer(id string) error {
	return exec.Command("docker", "restart", id).Run()
}

// RemoveContainer removes a container
func (c *Client) RemoveContainer(id string, force bool) error {
	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, id)
	return exec.Command("docker", args...).Run()
}

// GetContainerLogs gets container logs
func (c *Client) GetContainerLogs(id string, tail int, follow bool) (string, error) {
	args := []string{"logs"}
	if tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", tail))
	}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, id)

	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}

	return string(out), nil
}

// ExecInContainer executes a command in a container
func (c *Client) ExecInContainer(id string, cmd []string, interactive bool) error {
	args := []string{"exec"}
	if interactive {
		args = append(args, "-it")
	}
	args = append(args, id)
	args = append(args, cmd...)

	command := exec.Command("docker", args...)
	command.Stdin = nil
	command.Stdout = nil
	command.Stderr = nil

	return command.Run()
}

// GetContainerStats gets container statistics
func (c *Client) GetContainerStats(id string) (*ContainerStats, error) {
	out, err := exec.Command("docker", "stats", "--no-stream", "--format",
		"{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}",
		id).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	// Parse output (simplified)
	stats := &ContainerStats{}
	fields := strings.Split(strings.TrimSpace(string(out)), "\t")
	if len(fields) >= 1 {
		cpuStr := strings.TrimSuffix(fields[0], "%")
		stats.CPUPercent, _ = parseFloat(cpuStr)
	}

	return stats, nil
}

// ListImages lists all images
func (c *Client) ListImages() ([]Image, error) {
	out, err := exec.Command("docker", "images", "--format",
		"{{.ID}}\t{{.Repository}}\t{{.Tag}}\t{{.CreatedAt}}\t{{.Size}}").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	var images []Image
	scanner := bufio.NewScanner(strings.NewReader(string(out)))

	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) >= 5 {
			images = append(images, Image{
				ID:         fields[0],
				Repository: fields[1],
				Tag:        fields[2],
				Created:    fields[3],
				Size:       fields[4],
			})
		}
	}

	return images, nil
}

// PullImage pulls an image
func (c *Client) PullImage(name string) error {
	cmd := exec.Command("docker", "pull", name)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// RemoveImage removes an image
func (c *Client) RemoveImage(id string, force bool) error {
	args := []string{"rmi"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, id)
	return exec.Command("docker", args...).Run()
}

// BuildImage builds an image
func (c *Client) BuildImage(path string, tag string, dockerfile string) error {
	args := []string{"build", "-t", tag}
	if dockerfile != "" {
		args = append(args, "-f", dockerfile)
	}
	args = append(args, path)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// ListNetworks lists all networks
func (c *Client) ListNetworks() ([]Network, error) {
	out, err := exec.Command("docker", "network", "ls", "--format",
		"{{.ID}}\t{{.Name}}\t{{.Driver}}\t{{.Scope}}").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	var networks []Network
	scanner := bufio.NewScanner(strings.NewReader(string(out)))

	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) >= 4 {
			networks = append(networks, Network{
				ID:     fields[0],
				Name:   fields[1],
				Driver: fields[2],
				Scope:  fields[3],
			})
		}
	}

	return networks, nil
}

// CreateNetwork creates a network
func (c *Client) CreateNetwork(name string, driver string) error {
	args := []string{"network", "create"}
	if driver != "" {
		args = append(args, "-d", driver)
	}
	args = append(args, name)
	return exec.Command("docker", args...).Run()
}

// RemoveNetwork removes a network
func (c *Client) RemoveNetwork(name string) error {
	return exec.Command("docker", "network", "rm", name).Run()
}

// ListVolumes lists all volumes
func (c *Client) ListVolumes() ([]Volume, error) {
	out, err := exec.Command("docker", "volume", "ls", "--format",
		"{{.Name}}\t{{.Driver}}\t{{.Mountpoint}}").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	var volumes []Volume
	scanner := bufio.NewScanner(strings.NewReader(string(out)))

	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) >= 3 {
			volumes = append(volumes, Volume{
				Name:       fields[0],
				Driver:     fields[1],
				Mountpoint: fields[2],
			})
		}
	}

	return volumes, nil
}

// CreateVolume creates a volume
func (c *Client) CreateVolume(name string) error {
	return exec.Command("docker", "volume", "create", name).Run()
}

// RemoveVolume removes a volume
func (c *Client) RemoveVolume(name string, force bool) error {
	args := []string{"volume", "rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, name)
	return exec.Command("docker", args...).Run()
}

// RunContainer runs a new container
func (c *Client) RunContainer(image string, name string, options map[string]string) (string, error) {
	args := []string{"run", "-d"}

	if name != "" {
		args = append(args, "--name", name)
	}

	for key, value := range options {
		switch key {
		case "port", "p":
			args = append(args, "-p", value)
		case "volume", "v":
			args = append(args, "-v", value)
		case "env", "e":
			args = append(args, "-e", value)
		case "network":
			args = append(args, "--network", value)
		case "restart":
			args = append(args, "--restart", value)
		}
	}

	args = append(args, image)

	out, err := exec.Command("docker", args...).Output()
	if err != nil {
		return "", fmt.Errorf("failed to run container: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// DockerCompose operations
type ComposeClient struct {
	workDir string
}

// NewComposeClient creates a new Docker Compose client
func NewComposeClient(workDir string) *ComposeClient {
	return &ComposeClient{workDir: workDir}
}

// Up starts services
func (c *ComposeClient) Up(detach bool) error {
	args := []string{"compose", "up"}
	if detach {
		args = append(args, "-d")
	}

	cmd := exec.Command("docker", args...)
	cmd.Dir = c.workDir
	return cmd.Run()
}

// Down stops services
func (c *ComposeClient) Down(removeVolumes bool) error {
	args := []string{"compose", "down"}
	if removeVolumes {
		args = append(args, "-v")
	}

	cmd := exec.Command("docker", args...)
	cmd.Dir = c.workDir
	return cmd.Run()
}

// Logs gets service logs
func (c *ComposeClient) Logs(service string, follow bool) (string, error) {
	args := []string{"compose", "logs"}
	if follow {
		args = append(args, "-f")
	}
	if service != "" {
		args = append(args, service)
	}

	cmd := exec.Command("docker", args...)
	cmd.Dir = c.workDir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// PS lists services
func (c *ComposeClient) PS() (string, error) {
	cmd := exec.Command("docker", "compose", "ps")
	cmd.Dir = c.workDir
	out, err := cmd.Output()
	return string(out), err
}

func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
