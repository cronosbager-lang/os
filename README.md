# MIXOS GO

**AI-Powered Operating System for Developers**

MIXOS GO is a Linux-based operating system with an integrated AI agent that assists with system operations, development tasks, and autonomous management.

## Features

- 🤖 **AI-Powered**: Built-in AI agent (mix-small-1.1b) for intelligent system assistance
- 📦 **Modern Package Manager**: mix-pkg with AI-assisted installation
- 🐳 **Container-Ready**: Docker integration out of the box
- 🛠️ **Developer-Focused**: Pre-configured development environment
- ⚡ **Fast Boot**: Optimized kernel and initramfs with AI-guided hardware detection

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        USER SPACE                           │
├─────────────────────────────────────────────────────────────┤
│  mix-cli (Go)  │  mix-installer (Go)  │  Developer Tools   │
├─────────────────────────────────────────────────────────────┤
│              MIXOS AI AGENT (Python + llama.cpp)            │
├─────────────────────────────────────────────────────────────┤
│                   PACKAGE MANAGER (mix-pkg)                 │
├─────────────────────────────────────────────────────────────┤
│                    SYSTEMD + DOCKER                         │
├─────────────────────────────────────────────────────────────┤
│              LINUX KERNEL (Custom Configuration)            │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Build from Source

```bash
# Install dependencies
make deps

# Build everything
make all

# Build ISO
make iso

# Test in QEMU
make test-qemu
```

## Components

| Component | Language | Description |
|-----------|----------|-------------|
| mix-cli | Go | Command-line interface |
| mix-pkg | Rust | Package manager |
| mix-agent | Python | Main AI agent |
| mix-agent-early | Go | Early boot AI (static) |
| mix-installer | Go | TUI installer |

## Requirements

### Build Host
- Linux x86_64
- GCC 13+ or Clang 17+
- Go 1.21+
- Rust 1.75+
- Python 3.11+

### Target System
- x86_64 CPU
- 4GB RAM minimum (8GB recommended)
- 20GB disk space

## License

MIT License
