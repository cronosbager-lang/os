# MixOS

**Custom Operating System with Go Init, OCaml Services, and Python Agent**

MixOS is a from-scratch operating system designed for deterministic builds and AI-powered system management.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         USERLAND                                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    User Tools (mix-*)                        │   │
│  │  ┌───────────┐ ┌─────────────┐ ┌───────────┐ ┌───────────┐  │   │
│  │  │ mix-cli   │ │mix-installer│ │  mix-pkg  │ │mix-agent- │  │   │
│  │  │ (Go)      │ │ (Go/TUI)    │ │  (Rust)   │ │early (Go) │  │   │
│  │  └─────┬─────┘ └──────┬──────┘ └─────┬─────┘ └───────────┘  │   │
│  │        └──────────────┼──────────────┘                       │   │
│  └───────────────────────┼──────────────────────────────────────┘   │
│                          │                                          │
│  /bin                    │  /svc                      /agent        │
│  ┌─────────────┐         │ ┌─────────────────────┐   ┌───────────┐  │
│  │  Go Init    │         │ │  OCaml Services     │   │  Python   │  │
│  │  (PID 1)    │         │ │  ├── broker         │   │  Agent    │  │
│  │             │         │ │  ├── pkgmgr         │   │           │  │
│  │  Supervisor │         │ │  ├── builder        │   │  AI/LLM   │  │
│  │  IPC Server │         │ │  ├── resolver       │   │  Tools    │  │
│  └──────┬──────┘         │ │  └── cache          │   └─────┬─────┘  │
│         │                │ └──────────┬──────────┘         │        │
│         │                │            │                     │        │
│         └────────────────┴────────────┼─────────────────────┘        │
│                                       │                              │
│  ┌────────────────────────────────────▼────────────────────────────┐│
│  │                    IPC Layer (Unix Socket)                      ││
│  │                    Protocol: JSON (length-prefixed)             ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  /store                                                             │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │              Content-Addressable Store                        │  │
│  │              (Artifacts, Packages, Cache)                     │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│                         SYSCALL ABI                                 │
├─────────────────────────────────────────────────────────────────────┤
│                      MIXOS KERNEL (Rust - Future)                   │
└─────────────────────────────────────────────────────────────────────┘
```

## Components

### User Tools (mix-*)

| Tool | Language | Description |
|------|----------|-------------|
| **mix-cli** | Go | Command-line interface for system management |
| **mix-installer** | Go | TUI-based system installer with AI assistance |
| **mix-pkg** | Rust | Package manager CLI with multiple backends |
| **mix-agent-early** | Go | Early boot agent for hardware detection |
| **mix-agent** | Python | Full AI agent with LLM integration |

### /bin - Go Init System
- **PID 1 process** - First userland process
- **Service supervisor** - Manages service lifecycle
- **IPC server** - Central message routing
- **Static binary** - Compiled with musl for minimal dependencies

### /svc - OCaml Services
| Service | Description |
|---------|-------------|
| broker | IPC message broker and routing |
| pkgmgr | Package manager internal service |
| builder | Deterministic build executor |
| resolver | Dependency graph resolution |
| cache | Content-addressable artifact cache |

### /agent - Python Agent
- AI-powered system assistant
- Tool execution framework
- Integration with OCaml services via IPC

### /store - Artifact Store
- Content-addressable storage
- Build artifacts
- Package cache
- Deterministic paths based on content hash

## Directory Structure

```
/
├── bin/                    # Go init system (PID 1)
│   ├── main.go
│   └── pkg/
│       ├── config/         # Configuration loader
│       ├── ipc/            # IPC server
│       ├── service/        # Service management
│       └── supervisor/     # Process supervision
│
├── svc/                    # OCaml services
│   ├── broker/             # IPC broker
│   ├── pkgmgr/             # Package manager service
│   ├── builder/            # Build executor
│   ├── resolver/           # Dependency resolver
│   └── cache/              # Artifact cache
│
├── agent/                  # Python agent IPC integration
│   ├── ipc/                # IPC client library
│   ├── service.py          # Agent service wrapper
│   └── run.sh              # Startup script
│
├── store/                  # Content-addressable store
│
├── etc/                    # Configuration
│   └── init.toml           # Init configuration
│
├── mix-agent/              # Python AI agent (full implementation)
│   └── mixos_agent/        # Agent core, tools, inference, etc.
│
├── mix-agent-early/        # Go early boot agent
│   ├── cmd/                # Commands (detect, analyze, init)
│   └── pkg/                # Hardware detection, boot, inference
│
├── mix-cli/                # Go CLI tool
│   ├── cmd/                # Commands (install, remove, agent, etc.)
│   └── pkg/                # IPC client, config, utils
│
├── mix-installer/          # Go TUI installer
│   ├── ai/                 # AI-assisted installation
│   ├── backend/            # Disk, filesystem, bootloader
│   └── ui/                 # Bubbletea TUI components
│
├── mix-pkg/                # Rust package manager
│   └── src/                # Backend, CLI, config, core
│
├── kernel/                 # Future: Rust kernel
│
└── docs/                   # Documentation
```

## IPC Protocol

### Message Format
```json
{
  "version": 1,
  "msg_type": "REQUEST",
  "msg_id": 12345,
  "source": "agent",
  "target": "pkgmgr",
  "method": "package",
  "payload": "...",
  "timestamp": 1704931200000
}
```

### Message Types
- `REQUEST` - RPC request
- `RESPONSE` - RPC response
- `EVENT` - Broadcast event
- `STREAM` - Streaming data

### Service Methods

**pkgmgr:**
- `package` - Install/remove/query packages

**builder:**
- `build` - Execute deterministic build
- `status` - Get build status

**resolver:**
- `resolve` - Resolve dependency graph

**cache:**
- `cache` - Get/put/delete cache entries

## Building

```bash
# Install dependencies
make deps

# Build everything
make all

# Build specific components
make build-init       # Go init
make build-services   # OCaml services
make build-agent      # Python agent

# Create distribution
make dist

# Run tests
make test
```

## Requirements

### Build Host
- Go 1.21+
- OCaml 4.14+ with opam
- Python 3.11+
- Rust 1.75+ (for mix-pkg)

### OCaml Dependencies
```
opam install lwt lwt_ppx yojson ppx_deriving ppx_deriving_yojson digestif
```

## Boot Flow

```
1. Kernel loads /bin/init
2. Init creates IPC socket at /run/mixos/ipc.sock
3. Init starts services in dependency order:
   - broker (IPC routing)
   - pkgmgr (package management)
   - builder (build execution)
   - resolver (dependency resolution)
   - cache (artifact caching)
   - agent (AI assistant)
4. Services register with broker
5. System ready for operation
```

## Configuration

See `etc/init.toml` for init system configuration.

## License

MIT License
