# MIXOS GO Developer Guide

## Overview

This guide covers how to develop, build, and contribute to MIXOS GO.

## Development Environment Setup

### Prerequisites

- Linux development machine (Ubuntu 22.04+ recommended)
- Git
- Go 1.21+
- Rust 1.75+
- Python 3.11+
- Docker (for testing)
- QEMU (for VM testing)

### Quick Setup

```bash
# Clone the repository
git clone https://github.com/mixos/mixos-go.git
cd mixos-go

# Run setup script
./scripts/dev/setup-dev-env.sh

# Build everything
make all
```

### Manual Setup

```bash
# Install dependencies (Ubuntu/Debian)
sudo apt-get update
sudo apt-get install -y \
    build-essential gcc g++ make cmake \
    git curl wget \
    python3 python3-pip python3-venv \
    golang-go \
    rustc cargo \
    flex bison bc libssl-dev libelf-dev \
    xorriso grub-pc-bin grub-efi-amd64-bin mtools \
    squashfs-tools dosfstools \
    qemu-system-x86 qemu-utils

# Setup Go modules
cd mix-cli && go mod download && cd ..
cd mix-installer && go mod download && cd ..
cd mix-agent-early && go mod download && cd ..

# Setup Rust
cd mix-pkg && cargo fetch && cd ..

# Setup Python
cd mix-agent
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
deactivate
cd ..
```

## Project Structure

```
mixos-go/
├── kernel/              # Kernel configuration and build
├── initramfs/           # Early boot system
├── rootfs/              # Root filesystem
├── mix-cli/             # CLI tool (Go)
├── mix-pkg/             # Package manager (Rust)
├── mix-agent/           # AI agent (Python)
├── mix-agent-early/     # Early boot agent (Go)
├── mix-installer/       # TUI installer (Go)
├── iso/                 # ISO building
├── packages/            # Package definitions
├── configs/             # Configuration templates
├── scripts/             # Build and utility scripts
├── tests/               # Test suite
├── docs/                # Documentation
└── Makefile             # Build orchestration
```

## Building Components

### Build All

```bash
make all
```

### Build Individual Components

```bash
# Kernel
make kernel

# CLI tool
make mix-cli

# Package manager
make mix-pkg

# AI agent
make mix-agent

# Early boot agent
make mix-agent-early

# Installer
make mix-installer

# Initramfs
make initramfs

# RootFS
make rootfs

# ISO
make iso
```

### Build Options

```bash
# Clean build
make clean && make all

# Skip kernel (faster)
./scripts/build/build-all.sh --skip-kernel

# Verbose output
make VERBOSE=1 all
```

## Component Development

### mix-cli (Go)

Location: `mix-cli/`

```bash
cd mix-cli

# Build
go build -o mix-cli .

# Run tests
go test ./...

# Run with verbose
go run . --help
```

Adding a new command:

```go
// cmd/mycommand.go
package cmd

import "github.com/spf13/cobra"

var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "Description",
    Run: func(cmd *cobra.Command, args []string) {
        // Implementation
    },
}

func init() {
    rootCmd.AddCommand(myCmd)
}
```

### mix-pkg (Rust)

Location: `mix-pkg/`

```bash
cd mix-pkg

# Build
cargo build --release

# Run tests
cargo test

# Run
cargo run -- --help
```

Adding a new command:

```rust
// src/cli/mycommand.rs
use anyhow::Result;

pub fn run() -> Result<()> {
    // Implementation
    Ok(())
}
```

### mix-agent (Python)

Location: `mix-agent/`

```bash
cd mix-agent

# Setup venv
python3 -m venv venv
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt
pip install -r requirements-dev.txt

# Run tests
pytest tests/

# Run agent
python -m mixos_agent start --foreground

# Run API server
uvicorn mixos_agent.api.server:app --reload
```

Adding a new tool:

```python
# mixos_agent/tools/mytool.py
from dataclasses import dataclass, field
from typing import List
from .base import Tool, ToolParameter, SafetyLevel

@dataclass
class MyTool(Tool):
    name: str = "my_tool"
    description: str = "Description of my tool"
    parameters: List[ToolParameter] = field(default_factory=lambda: [
        ToolParameter(
            name="param1",
            type="string",
            description="Parameter description",
            required=True,
        ),
    ])
    safety_level: SafetyLevel = SafetyLevel.SAFE
    
    def execute(self, param1: str) -> str:
        # Implementation
        return f"Result: {param1}"
```

Register in `mixos_agent/tools/registry.py`:

```python
from .mytool import MyTool

class ToolRegistry:
    def _register_default_tools(self):
        # ... existing tools ...
        self.register(MyTool())
```

### mix-installer (Go)

Location: `mix-installer/`

```bash
cd mix-installer

# Build
go build -o mix-installer .

# Run
./mix-installer
```

Adding a new screen:

```go
// ui/screens/myscreen.go
package screens

import (
    tea "github.com/charmbracelet/bubbletea"
)

type MyScreen struct {
    // fields
}

func NewMyScreen() *MyScreen {
    return &MyScreen{}
}

func (s *MyScreen) Init() tea.Cmd {
    return nil
}

func (s *MyScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle input
    return s, nil
}

func (s *MyScreen) View() string {
    return "My Screen Content"
}
```

## Testing

### Run All Tests

```bash
./tests/run-all-tests.sh
```

### Unit Tests

```bash
# Go tests
cd mix-cli && go test ./...
cd mix-installer && go test ./...
cd mix-agent-early && go test ./...

# Rust tests
cd mix-pkg && cargo test

# Python tests
cd mix-agent && pytest tests/
```

### Integration Tests

```bash
./tests/integration/test_mix_cli.sh
python3 ./tests/integration/test_mix_agent.py
```

### QEMU Tests

```bash
# Boot test
./tests/qemu/test-boot.sh

# Installation test
./tests/qemu/test-install.sh
```

### Manual Testing

```bash
# Start QEMU with ISO
./scripts/dev/start-qemu.sh

# With UEFI
./scripts/dev/start-qemu.sh --uefi

# With disk for installation testing
./scripts/dev/start-qemu.sh --disk 20G
```

## Code Style

### Go

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Use `golint` for linting

```bash
gofmt -w .
golint ./...
```

### Rust

- Follow [Rust Style Guide](https://doc.rust-lang.org/style-guide/)
- Use `rustfmt` for formatting
- Use `clippy` for linting

```bash
cargo fmt
cargo clippy
```

### Python

- Follow [PEP 8](https://pep8.org/)
- Use `black` for formatting
- Use `flake8` for linting

```bash
black .
flake8 .
```

## Contributing

### Workflow

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make changes
4. Run tests: `./tests/run-all-tests.sh`
5. Commit: `git commit -m "Add my feature"`
6. Push: `git push origin feature/my-feature`
7. Create a Pull Request

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new command for VM management
fix: resolve package installation issue
docs: update user guide
test: add integration tests for agent
refactor: simplify tool registry
```

### Pull Request Guidelines

- Include tests for new features
- Update documentation as needed
- Ensure all tests pass
- Keep changes focused and atomic
- Respond to review feedback

## Debugging

### Agent Debugging

```bash
# Enable debug logging
export MIXOS_LOG_LEVEL=debug
python -m mixos_agent start --foreground

# Check logs
journalctl -u mixos-agent -f
```

### Kernel Debugging

```bash
# Boot with debug options
./scripts/dev/start-qemu.sh --debug

# Serial console
./scripts/dev/start-qemu.sh --serial --no-graphics
```

### Build Debugging

```bash
# Verbose make
make VERBOSE=1 mix-cli

# Check build logs
cat build/kernel-build.log
```

## Release Process

1. Update version numbers
2. Update CHANGELOG.md
3. Run full test suite
4. Build release artifacts
5. Create GitHub release
6. Update documentation

```bash
# Build release
VERSION=1.0.0 make all

# Create release ISO
VERSION=1.0.0 make iso
```

## Resources

- [Go Documentation](https://golang.org/doc/)
- [Rust Book](https://doc.rust-lang.org/book/)
- [Python Documentation](https://docs.python.org/3/)
- [Bubbletea](https://github.com/charmbracelet/bubbletea)
- [Cobra](https://github.com/spf13/cobra)
- [FastAPI](https://fastapi.tiangolo.com/)
- [llama.cpp](https://github.com/ggerganov/llama.cpp)
