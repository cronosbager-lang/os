# MixOS Architecture

## Overview

MixOS is a custom operating system designed with three main language runtimes:
- **Go** - Init system and supervisor (static binary)
- **OCaml** - Core system services (type-safe, deterministic)
- **Python** - AI agent and tooling

## System Layers

```
┌─────────────────────────────────────────────────────────────────────┐
│                         APPLICATION LAYER                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │
│  │  mix-cli    │  │ mix-install │  │  User Apps  │                 │
│  │  (Go)       │  │  (Go)       │  │             │                 │
│  └─────────────┘  └─────────────┘  └─────────────┘                 │
├─────────────────────────────────────────────────────────────────────┤
│                         AGENT LAYER                                 │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                    Python Agent                               │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │  │
│  │  │  Brain   │  │ Executor │  │  Tools   │  │  Memory  │      │  │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘      │  │
│  └───────────────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────────┤
│                         SERVICE LAYER                               │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    OCaml Services                           │    │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌───────┐ │    │
│  │  │ Broker  │ │ PkgMgr  │ │ Builder │ │Resolver │ │ Cache │ │    │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └───────┘ │    │
│  └─────────────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────────────┤
│                         IPC LAYER                                   │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │              Unix Domain Socket + JSON/Protobuf               │  │
│  └───────────────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────────┤
│                         INIT LAYER                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                    Go Init (PID 1)                            │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │  │
│  │  │ Config   │  │Supervisor│  │IPC Server│  │  Signals │      │  │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘      │  │
│  └───────────────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────────┤
│                         STORE LAYER                                 │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │              Content-Addressable Store (/store)               │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐                    │  │
│  │  │ Packages │  │ Artifacts│  │  Cache   │                    │  │
│  │  └──────────┘  └──────────┘  └──────────┘                    │  │
│  └───────────────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────────┤
│                         KERNEL (Future)                             │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                    Rust Kernel                                │  │
│  │  Memory │ Process │ Scheduler │ VFS │ Drivers │ Syscalls     │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

## Component Details

### Go Init System

The init system is the first userland process (PID 1) and is responsible for:

1. **Configuration Loading** - Parse `/etc/mixos/init.toml`
2. **IPC Server** - Create and manage Unix socket at `/run/mixos/ipc.sock`
3. **Service Supervision** - Start, stop, and monitor services
4. **Signal Handling** - Handle SIGTERM, SIGINT, SIGHUP

```go
// Service lifecycle
type ServiceState int
const (
    StateStopped ServiceState = iota
    StateStarting
    StateRunning
    StateStopping
    StateFailed
)
```

### OCaml Services

#### Broker
Central message routing service:
- Client registration
- Message forwarding
- Event broadcasting
- Subscription management

#### Package Manager (pkgmgr)
Package operations:
- Install packages to `/store`
- Remove packages
- Query package database
- List installed packages

#### Build Executor (builder)
Deterministic build execution:
- Isolated build environment
- Reproducible builds
- Content-addressed output
- Build logging

#### Dependency Resolver (resolver)
Dependency graph management:
- Topological sorting
- Version constraint solving
- Cycle detection
- Install order calculation

#### Artifact Cache (cache)
Content-addressable caching:
- Get/put/delete operations
- TTL-based expiration
- Garbage collection
- Statistics

### Python Agent

AI-powered system assistant:
- Natural language interface
- Tool execution
- Integration with OCaml services
- Memory and context management

## IPC Protocol

### Message Structure

```
┌────────────────────────────────────────┐
│  Length Prefix (4 bytes, big-endian)   │
├────────────────────────────────────────┤
│  JSON/Protobuf Payload                 │
│  {                                     │
│    "version": 1,                       │
│    "msg_type": "REQUEST",              │
│    "msg_id": 12345,                    │
│    "source": "agent",                  │
│    "target": "pkgmgr",                 │
│    "method": "package",                │
│    "payload": "...",                   │
│    "timestamp": 1704931200000          │
│  }                                     │
└────────────────────────────────────────┘
```

### Communication Flow

```
┌────────┐     ┌────────┐     ┌────────┐
│ Agent  │     │ Broker │     │ PkgMgr │
└───┬────┘     └───┬────┘     └───┬────┘
    │              │              │
    │  REQUEST     │              │
    │─────────────>│              │
    │              │  FORWARD     │
    │              │─────────────>│
    │              │              │
    │              │  RESPONSE    │
    │              │<─────────────│
    │  RESPONSE    │              │
    │<─────────────│              │
    │              │              │
```

## Store Layout

```
/store/
├── <hash1>/              # Package or artifact
│   ├── bin/
│   ├── lib/
│   └── share/
├── <hash2>/
│   └── ...
├── cache/                # Artifact cache
│   ├── ab/
│   │   └── <hash>/
│   └── cd/
│       └── <hash>/
└── db/                   # Package database
    └── packages.db
```

## Boot Sequence

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. Kernel loads /bin/init                                       │
├─────────────────────────────────────────────────────────────────┤
│ 2. Init reads /etc/mixos/init.toml                              │
├─────────────────────────────────────────────────────────────────┤
│ 3. Init creates /run/mixos/ipc.sock                             │
├─────────────────────────────────────────────────────────────────┤
│ 4. Init starts IPC server                                       │
├─────────────────────────────────────────────────────────────────┤
│ 5. Init resolves service dependencies                           │
├─────────────────────────────────────────────────────────────────┤
│ 6. Init starts services in order:                               │
│    a. broker                                                    │
│    b. pkgmgr, builder, resolver, cache (parallel)               │
│    c. agent                                                     │
├─────────────────────────────────────────────────────────────────┤
│ 7. Services register with broker                                │
├─────────────────────────────────────────────────────────────────┤
│ 8. System ready                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Deterministic Builds

The builder service ensures reproducible builds:

1. **Isolated Environment**
   - Clean build directory
   - Controlled environment variables
   - Fixed timestamps (SOURCE_DATE_EPOCH=1)

2. **Content Addressing**
   - Output hash based on content
   - Store path: `/store/<sha256-hash>/`

3. **Build Script**
   ```bash
   # Environment
   HOME=/tmp/build-xxx
   TMPDIR=/tmp/build-xxx/tmp
   SOURCE_DATE_EPOCH=1
   TZ=UTC
   LC_ALL=C
   
   # Execute
   cd /tmp/build-xxx/src
   sh build.sh
   ```

## Security Model

1. **Process Isolation** - Each service runs as separate process
2. **IPC Authentication** - Services register with broker
3. **Store Integrity** - Content-addressed storage prevents tampering
4. **Sandboxed Builds** - Isolated build environment

## Future: Rust Kernel

Planned kernel features:
- Custom syscall ABI
- Memory management (4-level paging)
- Process/thread management
- Basic device drivers
- VFS layer
