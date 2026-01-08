# MIXOS GO Architecture

## Overview

MIXOS GO is an AI-powered operating system designed for developers. It integrates a local AI agent that assists with system administration, development tasks, and autonomous operations.

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        USER SPACE                           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  mix-cli     │  │ mix-installer│  │  Developer   │      │
│  │  (Go)        │  │  (Bubbletea) │  │  Tools       │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                  │                  │             │
│         └──────────────────┼──────────────────┘             │
│                            │                                │
│  ┌─────────────────────────▼──────────────────────────┐    │
│  │           MIXOS AI AGENT (Python)                  │    │
│  │  ┌──────────────────────────────────────────────┐  │    │
│  │  │  Mix-Small Model (TinyLlama fine-tuned)      │  │    │
│  │  │  - Tool Calling Engine                       │  │    │
│  │  │  - System Operations                         │  │    │
│  │  │  - Context Management                        │  │    │
│  │  └──────────────────────────────────────────────┘  │    │
│  └─────────────────────────┬──────────────────────────┘    │
│                            │                                │
│  ┌─────────────────────────▼──────────────────────────┐    │
│  │         PACKAGE MANAGER (mix-pkg)                  │    │
│  │  - Repository Management                           │    │
│  │  - Dependency Resolution                           │    │
│  └─────────────────────────┬──────────────────────────┘    │
│                            │                                │
├────────────────────────────┼────────────────────────────────┤
│                     SYSTEM LAYER                            │
├────────────────────────────┼────────────────────────────────┤
│  ┌─────────────────────────▼──────────────────────────┐    │
│  │              SYSTEMD (Init System)                 │    │
│  │  - mixos-agent.service                             │    │
│  │  - docker.service                                  │    │
│  └─────────────────────────┬──────────────────────────┘    │
│                            │                                │
│  ┌─────────────────────────▼──────────────────────────┐    │
│  │         DOCKER / CONTAINERD                        │    │
│  └─────────────────────────┬──────────────────────────┘    │
│                            │                                │
├────────────────────────────┼────────────────────────────────┤
│                      KERNEL SPACE                           │
├────────────────────────────┼────────────────────────────────┤
│  ┌─────────────────────────▼──────────────────────────┐    │
│  │     LINUX KERNEL (Custom Configuration)            │    │
│  │  - Optimized for containers                        │    │
│  │  - cgroups v2, namespaces                          │    │
│  └────────────────────────────────────────────────────┘    │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                    EARLY BOOT (Initramfs)                   │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────┐      │
│  │     AI AGENT EARLY (Go - Static Binary)          │      │
│  │  - Hardware Detection                             │      │
│  │  - Module Loading                                 │      │
│  │  - Root FS Detection                              │      │
│  └───────────────────┬──────────────────────────────┘      │
│                      │                                      │
│  ┌───────────────────▼──────────────────────────────┐      │
│  │  BUSYBOX (Minimal Unix Tools)                     │      │
│  └───────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

## Components

### 1. mix-cli (Go)

The command-line interface for MIXOS. Provides commands for:
- System management (`mix status`, `mix update`)
- Package operations (`mix install`, `mix remove`)
- AI agent control (`mix agent chat`, `mix agent execute`)
- VM and container management

### 2. mix-pkg (Rust)

The package manager for MIXOS. Features:
- Multiple backend support (ALPM, dpkg, native)
- Dependency resolution
- Repository management
- Package verification

### 3. mix-agent (Python)

The AI agent that powers intelligent system operations:
- **Core**: Agent loop, brain (reasoning), executor, planner
- **Inference**: llama.cpp integration for local LLM
- **Tools**: System, developer, network, and MIXOS-specific tools
- **Memory**: Context management, session tracking, vector store
- **Safety**: Command validation, sandboxing, audit logging
- **API**: FastAPI server with WebSocket support

### 4. mix-agent-early (Go)

Lightweight AI agent for early boot:
- Hardware detection
- Kernel module loading
- Root filesystem detection
- Emergency recovery assistance

### 5. mix-installer (Go)

TUI-based installer using Bubbletea:
- Autonomous installation mode
- AI-guided partitioning
- Hardware recommendations
- Post-install configuration

## Boot Flow

1. **BIOS/UEFI** → Hardware initialization
2. **GRUB** → Bootloader, kernel selection
3. **Kernel** → Linux kernel loads
4. **Initramfs** → Early userspace
   - mix-agent-early starts
   - Hardware detection
   - Module loading
   - Root filesystem mounting
5. **systemd** → Init system
   - Services start
   - mixos-agent.service launches
6. **Ready** → System operational

## AI Agent Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    AI AGENT                              │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────┐    ┌──────────────┐                   │
│  │    Brain     │◄──►│   Planner    │                   │
│  │  (Reasoning) │    │ (Task Plans) │                   │
│  └──────┬───────┘    └──────────────┘                   │
│         │                                                │
│         ▼                                                │
│  ┌──────────────┐    ┌──────────────┐                   │
│  │  Inference   │    │   Executor   │                   │
│  │  (LLM)       │    │ (Run Tools)  │                   │
│  └──────────────┘    └──────┬───────┘                   │
│                             │                            │
│  ┌──────────────┐    ┌──────▼───────┐                   │
│  │   Memory     │    │    Tools     │                   │
│  │  (Context)   │    │  (Registry)  │                   │
│  └──────────────┘    └──────────────┘                   │
│                                                          │
│  ┌──────────────┐    ┌──────────────┐                   │
│  │   Safety     │    │     API      │                   │
│  │ (Validation) │    │  (FastAPI)   │                   │
│  └──────────────┘    └──────────────┘                   │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Kernel | Linux 6.x | Operating system core |
| Init | systemd | Service management |
| mix-cli | Go + Cobra | CLI interface |
| mix-pkg | Rust + Clap | Package management |
| mix-agent | Python + FastAPI | AI agent |
| mix-installer | Go + Bubbletea | TUI installer |
| AI Model | TinyLlama + llama.cpp | Local inference |
| Containers | Docker + containerd | Container runtime |

## Security Model

1. **Safety Validator**: Blocks dangerous commands
2. **Permission System**: Fine-grained access control
3. **Sandboxing**: Isolated execution environment
4. **Audit Logging**: All actions logged
5. **Confirmation**: Destructive operations require approval

## Directory Structure

```
/
├── etc/
│   └── mixos/
│       ├── agent.toml      # Agent configuration
│       ├── repositories.toml
│       └── system.toml
├── opt/
│   └── mixos/
│       ├── agent/          # Python agent
│       ├── ai/
│       │   └── model/      # AI models
│       └── scripts/
├── usr/
│   └── local/
│       └── bin/
│           ├── mix-cli
│           ├── mix-pkg
│           └── mix-installer
└── var/
    ├── lib/
    │   └── mixos/
    │       └── agent/      # Agent data
    └── log/
        └── mixos/          # Logs
```
