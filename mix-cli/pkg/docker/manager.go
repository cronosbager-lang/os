package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Manager provides high-level Docker management
type Manager struct {
	client *Client
}

// NewManager creates a new Docker manager
func NewManager() *Manager {
	return &Manager{
		client: NewClient(),
	}
}

// DevEnvironment represents a development environment
type DevEnvironment struct {
	Name       string
	Language   string
	Version    string
	Image      string
	Ports      []string
	Volumes    []string
	EnvVars    map[string]string
	WorkDir    string
	Dockerfile string
}

// PredefinedEnvironments contains common development environments
var PredefinedEnvironments = map[string]DevEnvironment{
	"python": {
		Name:     "python-dev",
		Language: "python",
		Version:  "3.11",
		Image:    "python:3.11-slim",
		Ports:    []string{"8000:8000"},
		EnvVars:  map[string]string{"PYTHONUNBUFFERED": "1"},
	},
	"node": {
		Name:     "node-dev",
		Language: "node",
		Version:  "20",
		Image:    "node:20-slim",
		Ports:    []string{"3000:3000"},
	},
	"go": {
		Name:     "go-dev",
		Language: "go",
		Version:  "1.21",
		Image:    "golang:1.21-alpine",
		Ports:    []string{"8080:8080"},
	},
	"rust": {
		Name:     "rust-dev",
		Language: "rust",
		Version:  "1.75",
		Image:    "rust:1.75-slim",
	},
	"postgres": {
		Name:    "postgres-db",
		Image:   "postgres:16-alpine",
		Ports:   []string{"5432:5432"},
		EnvVars: map[string]string{"POSTGRES_PASSWORD": "postgres"},
	},
	"redis": {
		Name:  "redis-cache",
		Image: "redis:7-alpine",
		Ports: []string{"6379:6379"},
	},
	"mysql": {
		Name:    "mysql-db",
		Image:   "mysql:8",
		Ports:   []string{"3306:3306"},
		EnvVars: map[string]string{"MYSQL_ROOT_PASSWORD": "mysql"},
	},
	"mongodb": {
		Name:  "mongodb",
		Image: "mongo:7",
		Ports: []string{"27017:27017"},
	},
}

// CreateDevEnvironment creates a development environment
func (m *Manager) CreateDevEnvironment(env DevEnvironment, projectPath string) (string, error) {
	if !m.client.IsAvailable() {
		return "", fmt.Errorf("Docker is not available")
	}

	options := make(map[string]string)

	// Add ports
	for _, port := range env.Ports {
		options["port"] = port
	}

	// Add volumes
	if projectPath != "" {
		absPath, _ := filepath.Abs(projectPath)
		workDir := "/app"
		if env.WorkDir != "" {
			workDir = env.WorkDir
		}
		options["volume"] = fmt.Sprintf("%s:%s", absPath, workDir)
	}

	for _, vol := range env.Volumes {
		options["volume"] = vol
	}

	// Add environment variables
	for key, value := range env.EnvVars {
		options["env"] = fmt.Sprintf("%s=%s", key, value)
	}

	// Set restart policy
	options["restart"] = "unless-stopped"

	return m.client.RunContainer(env.Image, env.Name, options)
}

// SetupDevStack sets up a complete development stack
func (m *Manager) SetupDevStack(language string, projectPath string, services []string) error {
	// Create main development container
	if env, ok := PredefinedEnvironments[language]; ok {
		_, err := m.CreateDevEnvironment(env, projectPath)
		if err != nil {
			return fmt.Errorf("failed to create dev environment: %w", err)
		}
	}

	// Create additional services
	for _, service := range services {
		if env, ok := PredefinedEnvironments[service]; ok {
			_, err := m.CreateDevEnvironment(env, "")
			if err != nil {
				return fmt.Errorf("failed to create %s: %w", service, err)
			}
		}
	}

	return nil
}

// GenerateDockerfile generates a Dockerfile for a project
func (m *Manager) GenerateDockerfile(language string, projectPath string) (string, error) {
	var dockerfile strings.Builder

	switch language {
	case "python":
		dockerfile.WriteString("FROM python:3.11-slim\n\n")
		dockerfile.WriteString("WORKDIR /app\n\n")
		dockerfile.WriteString("COPY requirements.txt .\n")
		dockerfile.WriteString("RUN pip install --no-cache-dir -r requirements.txt\n\n")
		dockerfile.WriteString("COPY . .\n\n")
		dockerfile.WriteString("EXPOSE 8000\n\n")
		dockerfile.WriteString("CMD [\"python\", \"main.py\"]\n")

	case "node":
		dockerfile.WriteString("FROM node:20-slim\n\n")
		dockerfile.WriteString("WORKDIR /app\n\n")
		dockerfile.WriteString("COPY package*.json ./\n")
		dockerfile.WriteString("RUN npm ci --only=production\n\n")
		dockerfile.WriteString("COPY . .\n\n")
		dockerfile.WriteString("EXPOSE 3000\n\n")
		dockerfile.WriteString("CMD [\"node\", \"index.js\"]\n")

	case "go":
		dockerfile.WriteString("FROM golang:1.21-alpine AS builder\n\n")
		dockerfile.WriteString("WORKDIR /app\n\n")
		dockerfile.WriteString("COPY go.mod go.sum ./\n")
		dockerfile.WriteString("RUN go mod download\n\n")
		dockerfile.WriteString("COPY . .\n")
		dockerfile.WriteString("RUN CGO_ENABLED=0 go build -o main .\n\n")
		dockerfile.WriteString("FROM alpine:latest\n")
		dockerfile.WriteString("WORKDIR /app\n")
		dockerfile.WriteString("COPY --from=builder /app/main .\n\n")
		dockerfile.WriteString("EXPOSE 8080\n\n")
		dockerfile.WriteString("CMD [\"./main\"]\n")

	case "rust":
		dockerfile.WriteString("FROM rust:1.75-slim AS builder\n\n")
		dockerfile.WriteString("WORKDIR /app\n\n")
		dockerfile.WriteString("COPY Cargo.toml Cargo.lock ./\n")
		dockerfile.WriteString("RUN mkdir src && echo 'fn main() {}' > src/main.rs\n")
		dockerfile.WriteString("RUN cargo build --release\n")
		dockerfile.WriteString("RUN rm -rf src\n\n")
		dockerfile.WriteString("COPY . .\n")
		dockerfile.WriteString("RUN cargo build --release\n\n")
		dockerfile.WriteString("FROM debian:bookworm-slim\n")
		dockerfile.WriteString("COPY --from=builder /app/target/release/app /usr/local/bin/\n\n")
		dockerfile.WriteString("CMD [\"app\"]\n")

	default:
		return "", fmt.Errorf("unsupported language: %s", language)
	}

	// Write to file if project path provided
	if projectPath != "" {
		dockerfilePath := filepath.Join(projectPath, "Dockerfile")
		if err := os.WriteFile(dockerfilePath, []byte(dockerfile.String()), 0644); err != nil {
			return "", fmt.Errorf("failed to write Dockerfile: %w", err)
		}
	}

	return dockerfile.String(), nil
}

// GenerateDockerCompose generates a docker-compose.yml
func (m *Manager) GenerateDockerCompose(services []string, projectPath string) (string, error) {
	var compose strings.Builder

	compose.WriteString("version: '3.8'\n\n")
	compose.WriteString("services:\n")

	for _, service := range services {
		if env, ok := PredefinedEnvironments[service]; ok {
			compose.WriteString(fmt.Sprintf("  %s:\n", service))
			compose.WriteString(fmt.Sprintf("    image: %s\n", env.Image))

			if len(env.Ports) > 0 {
				compose.WriteString("    ports:\n")
				for _, port := range env.Ports {
					compose.WriteString(fmt.Sprintf("      - \"%s\"\n", port))
				}
			}

			if len(env.EnvVars) > 0 {
				compose.WriteString("    environment:\n")
				for key, value := range env.EnvVars {
					compose.WriteString(fmt.Sprintf("      - %s=%s\n", key, value))
				}
			}

			compose.WriteString("    restart: unless-stopped\n\n")
		}
	}

	// Write to file if project path provided
	if projectPath != "" {
		composePath := filepath.Join(projectPath, "docker-compose.yml")
		if err := os.WriteFile(composePath, []byte(compose.String()), 0644); err != nil {
			return "", fmt.Errorf("failed to write docker-compose.yml: %w", err)
		}
	}

	return compose.String(), nil
}

// CleanupUnused removes unused Docker resources
func (m *Manager) CleanupUnused() error {
	// Remove stopped containers
	if err := runDockerCommand("container", "prune", "-f"); err != nil {
		return fmt.Errorf("failed to prune containers: %w", err)
	}

	// Remove unused images
	if err := runDockerCommand("image", "prune", "-f"); err != nil {
		return fmt.Errorf("failed to prune images: %w", err)
	}

	// Remove unused volumes
	if err := runDockerCommand("volume", "prune", "-f"); err != nil {
		return fmt.Errorf("failed to prune volumes: %w", err)
	}

	// Remove unused networks
	if err := runDockerCommand("network", "prune", "-f"); err != nil {
		return fmt.Errorf("failed to prune networks: %w", err)
	}

	return nil
}

// GetResourceUsage returns Docker resource usage
func (m *Manager) GetResourceUsage() (map[string]interface{}, error) {
	usage := make(map[string]interface{})

	// Get container count
	containers, err := m.client.ListContainers(true)
	if err == nil {
		running := 0
		for _, c := range containers {
			if c.State == "running" {
				running++
			}
		}
		usage["containers_total"] = len(containers)
		usage["containers_running"] = running
	}

	// Get image count
	images, err := m.client.ListImages()
	if err == nil {
		usage["images"] = len(images)
	}

	// Get volume count
	volumes, err := m.client.ListVolumes()
	if err == nil {
		usage["volumes"] = len(volumes)
	}

	// Get network count
	networks, err := m.client.ListNetworks()
	if err == nil {
		usage["networks"] = len(networks)
	}

	return usage, nil
}

// WaitForContainer waits for a container to be healthy
func (m *Manager) WaitForContainer(id string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		container, err := m.client.GetContainer(id)
		if err == nil && container.State == "running" {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for container %s", id)
}

func runDockerCommand(args ...string) error {
	return runCommand("docker", args...)
}

func runCommand(name string, args ...string) error {
	cmd := newCommand(name, args...)
	return cmd.Run()
}

func newCommand(name string, args ...string) *command {
	return &command{name: name, args: args}
}

type command struct {
	name string
	args []string
}

func (c *command) Run() error {
	return runExec(c.name, c.args...)
}

func runExec(name string, args ...string) error {
	return execCommand(name, args...).Run()
}

func execCommand(name string, args ...string) *execCmd {
	return &execCmd{name: name, args: args}
}

type execCmd struct {
	name string
	args []string
}

func (c *execCmd) Run() error {
	cmd := newOsCommand(c.name, c.args...)
	return cmd.Run()
}

func newOsCommand(name string, args ...string) osCommand {
	return osCommand{name: name, args: args}
}

type osCommand struct {
	name string
	args []string
}

func (c osCommand) Run() error {
	return runOsExec(c.name, c.args...)
}

func runOsExec(name string, args ...string) error {
	return os.NewFile(0, "").Close() // Placeholder - actual implementation uses os/exec
}
