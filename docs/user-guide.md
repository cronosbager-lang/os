# MIXOS GO User Guide

## Introduction

Welcome to MIXOS GO, an AI-powered operating system designed for developers. This guide will help you get started with MIXOS and make the most of its features.

## Installation

### Requirements

- **CPU**: x86_64 processor (Intel/AMD)
- **RAM**: Minimum 2GB, recommended 4GB+
- **Storage**: Minimum 20GB
- **Boot**: UEFI or Legacy BIOS

### Installation Methods

#### 1. Autonomous Installation (Recommended)

Let the AI handle everything:

1. Boot from the MIXOS ISO
2. Select "MIXOS GO Installer"
3. Choose "Autonomous Installation"
4. Confirm the AI's recommendations
5. Wait for installation to complete
6. Reboot

#### 2. Guided Installation

Step-by-step with AI assistance:

1. Boot from the MIXOS ISO
2. Select "MIXOS GO Installer"
3. Choose "Guided Installation"
4. Follow the prompts for:
   - Disk selection
   - Partitioning
   - User creation
   - Package selection
5. Review and confirm
6. Wait for installation
7. Reboot

#### 3. Manual Installation

Full control over every step:

1. Boot into live environment
2. Use `mix-installer --manual` or partition manually
3. Mount filesystems
4. Install base system
5. Configure bootloader
6. Reboot

## Getting Started

### First Boot

After installation, MIXOS will:
1. Run first-boot configuration
2. Start the AI agent
3. Present a login prompt

Login with the credentials you created during installation.

### The mix Command

The `mix` command is your primary interface to MIXOS:

```bash
# Check system status
mix status

# Update package database
mix update

# Upgrade all packages
mix upgrade

# Install a package
mix install docker

# Remove a package
mix remove firefox

# Search for packages
mix search python
```

## AI Agent

### Starting a Chat

Talk to the AI agent:

```bash
mix agent chat
```

Example conversation:
```
You: Install Python and set up a virtual environment
AI: I'll help you set up Python. Let me install Python and create a virtual environment...
    ✓ Installing python3
    ✓ Installing python3-pip
    ✓ Creating virtual environment at ~/venv
    Done! Activate with: source ~/venv/bin/activate
```

### Executing Tasks

Run autonomous tasks:

```bash
# Setup development environment
mix agent execute "setup python development environment"

# Install and configure nginx
mix agent execute "install and configure nginx as reverse proxy"

# Create a new project
mix agent execute "create a FastAPI project called myapp"
```

### Agent Commands

```bash
mix agent start      # Start the agent
mix agent stop       # Stop the agent
mix agent status     # Check agent status
mix agent restart    # Restart the agent
mix agent chat       # Interactive chat
mix agent execute    # Run a task
mix agent config     # View/edit configuration
```

## Package Management

### Installing Packages

```bash
# Single package
mix install git

# Multiple packages
mix install git vim tmux

# Specific version
mix install python=3.11
```

### Removing Packages

```bash
mix remove firefox
```

### Searching

```bash
# Search by name
mix search docker

# Show package info
mix info docker
```

### Repositories

```bash
# List repositories
mix repo list

# Add a repository
mix repo add https://repo.example.com/packages

# Remove a repository
mix repo remove example
```

## Container Management

MIXOS includes Docker for container management:

```bash
# List containers
mix container list

# View logs
mix container logs <container-id>

# Execute command in container
mix container exec <container-id> bash
```

Or use Docker directly:

```bash
docker run -d nginx
docker ps
docker logs <container-id>
```

## Development Tools

### Setting Up Development Environments

```bash
# Setup Python
mix dev setup python

# Setup Node.js
mix dev setup node

# Setup Go
mix dev setup go

# Setup Rust
mix dev setup rust
```

### Project Initialization

```bash
# Initialize a Python project
mix dev init python

# Initialize a Node.js project
mix dev init node

# Initialize a Go project
mix dev init go
```

### AI-Assisted Development

Ask the AI to help with development tasks:

```bash
mix agent execute "create a REST API with FastAPI and PostgreSQL"
mix agent execute "set up CI/CD with GitHub Actions"
mix agent execute "containerize this application with Docker"
```

## Virtual Machines

MIXOS can manage virtual machines:

```bash
# Create a VM
mix vm create myvm --cpus 2 --memory 4G --disk 20G

# Start a VM
mix vm start myvm

# Stop a VM
mix vm stop myvm

# List VMs
mix vm list

# Destroy a VM
mix vm destroy myvm
```

## Configuration

### Agent Configuration

Edit `/etc/mixos/agent.toml`:

```toml
[agent]
name = "Mix Agent"
enabled = true

[model]
path = "/opt/mixos/ai/model/mix-small-1.1b-q4.gguf"
context_length = 4096
temperature = 0.7

[inference]
threads = 4

[safety]
require_confirmation = ["install_package", "remove_package"]
```

### System Configuration

Edit `/etc/mixos/system.toml`:

```toml
[system]
hostname = "mixos"
timezone = "UTC"
locale = "en_US.UTF-8"

[updates]
auto_update = true
update_schedule = "daily"
```

## Troubleshooting

### Agent Not Responding

```bash
# Check agent status
mix agent status

# Restart agent
mix agent restart

# Check logs
journalctl -u mixos-agent -f
```

### Package Installation Fails

```bash
# Update package database
mix update

# Clear cache
mix clean

# Try again
mix install <package>
```

### Boot Issues

1. Boot into safe mode from GRUB menu
2. Check logs: `journalctl -xb`
3. Ask the AI: `mix agent chat` → "Help me troubleshoot boot issues"

### Getting Help

```bash
# Command help
mix --help
mix agent --help

# Ask the AI
mix agent chat
> How do I configure networking?
```

## Tips and Tricks

### Aliases

MIXOS includes helpful aliases:

```bash
ai          # Shortcut for 'mix agent chat'
mix-status  # Shortcut for 'mix status'
mix-update  # Shortcut for 'mix update && mix upgrade'
```

### Keyboard Shortcuts

In `mix agent chat`:
- `Ctrl+C` - Cancel current operation
- `Ctrl+D` - Exit chat
- `↑/↓` - Navigate history

### Best Practices

1. **Keep Updated**: Run `mix update && mix upgrade` regularly
2. **Use AI**: Let the AI handle complex tasks
3. **Confirm Destructive Operations**: Always review before confirming
4. **Check Logs**: Use `journalctl` for troubleshooting
5. **Backup**: Regular backups before major changes

## Getting Help

- **Documentation**: https://mixos.dev/docs
- **Community**: https://community.mixos.dev
- **Issues**: https://github.com/mixos/mixos-go/issues
- **AI Chat**: `mix agent chat` - Ask the AI anything!
