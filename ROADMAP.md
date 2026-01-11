# MixOS Development Roadmap

## Overview

MixOS adalah custom operating system dengan arsitektur:
- **Go Init** - PID 1, service supervisor, IPC server
- **OCaml Services** - Type-safe system services (broker, pkgmgr, builder, resolver, cache)
- **Python Agent** - AI-powered system assistant
- **Rust Kernel** - Custom kernel (future)

---

## ✅ Phase 1: Foundation (COMPLETED)

### Milestone 1.1: Architecture Setup
- [x] Hapus komponen Linux-based (kernel, initramfs, rootfs)
- [x] Setup struktur direktori baru (`/bin`, `/svc`, `/agent`, `/store`)
- [x] Definisi IPC protocol (Unix socket + JSON)

### Milestone 1.2: Go Init System
- [x] Main init process (`bin/main.go`)
- [x] Configuration loader (`pkg/config`)
- [x] Service management (`pkg/service`)
- [x] Process supervisor (`pkg/supervisor`)
- [x] IPC server (`pkg/ipc`)

### Milestone 1.3: OCaml Services (Skeleton)
- [x] Broker - IPC message routing
- [x] PkgMgr - Package manager service
- [x] Builder - Build executor
- [x] Resolver - Dependency graph resolver
- [x] Cache - Artifact cache

### Milestone 1.4: Python Agent Integration
- [x] IPC client library (`agent/ipc/`)
- [x] Agent service wrapper (`agent/service.py`)
- [x] Startup script (`agent/run.sh`)

### Milestone 1.5: Mix Tools Integration
- [x] IPC client untuk mix-cli
- [x] Build targets untuk semua mix-* tools
- [x] Dokumentasi komponen

---

## 🔄 Phase 2: Medium Complexity (CURRENT)

### Milestone 2.1: Robust IPC System
**Goal:** Production-ready IPC dengan error handling dan reconnection

- [ ] **Connection Management**
  - [ ] Auto-reconnect pada connection loss
  - [ ] Connection pooling untuk high throughput
  - [ ] Heartbeat/keepalive mechanism
  - [ ] Graceful shutdown handling

- [ ] **Message Reliability**
  - [ ] Message acknowledgment
  - [ ] Request timeout handling
  - [ ] Retry logic dengan exponential backoff
  - [ ] Message queue untuk offline services

- [ ] **Protocol Enhancement**
  - [ ] Binary protocol option (MessagePack/Protobuf)
  - [ ] Streaming support untuk large payloads
  - [ ] Compression untuk large messages
  - [ ] Message versioning

### Milestone 2.2: Service Discovery & Health
**Goal:** Dynamic service management

- [ ] **Service Registry**
  - [ ] Service registration dengan capabilities
  - [ ] Service discovery by name/capability
  - [ ] Service metadata (version, status, endpoints)
  - [ ] Namespace support

- [ ] **Health Monitoring**
  - [ ] Health check endpoints per service
  - [ ] Liveness dan readiness probes
  - [ ] Resource monitoring (CPU, memory)
  - [ ] Automatic restart on failure

- [ ] **Load Balancing**
  - [ ] Round-robin untuk multiple instances
  - [ ] Weighted routing
  - [ ] Circuit breaker pattern

### Milestone 2.3: Package Manager Enhancement
**Goal:** Functional package management

- [ ] **Package Format**
  - [ ] Define package manifest format (TOML/JSON)
  - [ ] Package signing dan verification
  - [ ] Dependency specification format
  - [ ] Build recipe format

- [ ] **Repository System**
  - [ ] Local repository support
  - [ ] Remote repository fetching
  - [ ] Repository mirroring
  - [ ] Package index management

- [ ] **Installation Flow**
  - [ ] Dependency resolution integration
  - [ ] Pre/post install hooks
  - [ ] Atomic installation (rollback on failure)
  - [ ] File conflict detection

### Milestone 2.4: Build System Enhancement
**Goal:** Deterministic, reproducible builds

- [ ] **Sandbox Environment**
  - [ ] Isolated filesystem (chroot/namespace)
  - [ ] Network isolation
  - [ ] Resource limits (CPU, memory, disk)
  - [ ] Deterministic environment variables

- [ ] **Build Features**
  - [ ] Parallel builds
  - [ ] Incremental builds
  - [ ] Build caching (input-based)
  - [ ] Build logging dan artifacts

- [ ] **Reproducibility**
  - [ ] Fixed timestamps
  - [ ] Sorted file operations
  - [ ] Deterministic linking
  - [ ] Build verification (rebuild check)

### Milestone 2.5: Store Implementation
**Goal:** Content-addressable storage

- [ ] **Store Structure**
  ```
  /store/
  ├── objects/          # Content-addressed blobs
  │   ├── ab/cd1234...  # SHA256 prefix directories
  │   └── ...
  ├── packages/         # Installed packages
  │   ├── <hash>/       # Package contents
  │   └── ...
  ├── profiles/         # User profiles (symlink collections)
  │   └── default/
  └── gc-roots/         # GC protection
  ```

- [ ] **Operations**
  - [ ] Add object to store
  - [ ] Link package to profile
  - [ ] Garbage collection
  - [ ] Store verification/repair

---

## 📋 Phase 3: Advanced Features

### Milestone 3.1: Rust Kernel Foundation
**Goal:** Bootable minimal kernel

- [ ] **Boot**
  - [ ] Multiboot2 header
  - [ ] UEFI boot support
  - [ ] Early console output (VGA/serial)
  - [ ] GDT/IDT setup

- [ ] **Memory**
  - [ ] Physical memory manager
  - [ ] 4-level paging (x86_64)
  - [ ] Kernel heap allocator
  - [ ] Virtual memory manager

- [ ] **Interrupts**
  - [ ] IDT setup
  - [ ] Exception handlers
  - [ ] PIC/APIC initialization
  - [ ] Timer interrupt

### Milestone 3.2: Kernel Process Management
**Goal:** Multi-process support

- [ ] **Process**
  - [ ] Process structure
  - [ ] Process creation (fork-like)
  - [ ] Process termination
  - [ ] Process states

- [ ] **Scheduler**
  - [ ] Round-robin scheduler
  - [ ] Priority-based scheduling
  - [ ] Context switching
  - [ ] SMP support (future)

- [ ] **Syscall Interface**
  - [ ] Syscall handler (SYSCALL/SYSRET)
  - [ ] Basic syscalls (read, write, exit, fork, exec)
  - [ ] Syscall table
  - [ ] User/kernel transition

### Milestone 3.3: Kernel Filesystem
**Goal:** Basic VFS dan initramfs

- [ ] **VFS Layer**
  - [ ] Inode abstraction
  - [ ] File operations
  - [ ] Directory operations
  - [ ] Mount system

- [ ] **Initramfs**
  - [ ] CPIO parser
  - [ ] Ramfs implementation
  - [ ] Initial root mount

- [ ] **Device Files**
  - [ ] /dev/null, /dev/zero
  - [ ] /dev/console
  - [ ] Device major/minor numbers

### Milestone 3.4: Userland Bootstrap
**Goal:** Run Go init on custom kernel

- [ ] **ELF Loader**
  - [ ] ELF64 parser
  - [ ] Program loading
  - [ ] Dynamic linking (optional)
  - [ ] Static binary support

- [ ] **musl Integration**
  - [ ] Syscall mapping
  - [ ] Basic libc functions
  - [ ] Static linking

- [ ] **Init Handoff**
  - [ ] Kernel → init transition
  - [ ] Init environment setup
  - [ ] Service startup

---

## 📋 Phase 4: Production Ready

### Milestone 4.1: Networking
- [ ] Network device drivers (virtio-net)
- [ ] TCP/IP stack
- [ ] Socket syscalls
- [ ] DNS resolution

### Milestone 4.2: Storage
- [ ] Block device drivers (virtio-blk)
- [ ] Filesystem (ext4 read, custom write)
- [ ] Disk partitioning
- [ ] Boot from disk

### Milestone 4.3: Security
- [ ] User/group management
- [ ] File permissions
- [ ] Capability system
- [ ] Secure boot (optional)

### Milestone 4.4: AI Agent Full Integration
- [ ] Agent ↔ kernel communication
- [ ] System introspection tools
- [ ] Autonomous system management
- [ ] Natural language system control

---

## Timeline Estimate

| Phase | Duration | Status |
|-------|----------|--------|
| Phase 1: Foundation | 1-2 weeks | ✅ Complete |
| Phase 2: Medium Complexity | 3-4 weeks | 🔄 Current |
| Phase 3: Advanced Features | 6-8 weeks | 📋 Planned |
| Phase 4: Production Ready | 4-6 weeks | 📋 Planned |

---

## Technical Decisions

### Syscall ABI
```
Convention: x86_64 System V ABI
Syscall Number: RAX
Arguments: RDI, RSI, RDX, R10, R8, R9
Return: RAX (negative = -errno)
Clobbered: RCX, R11
```

### IPC Protocol
```
Transport: Unix Domain Socket
Encoding: JSON (length-prefixed)
Future: MessagePack/Protobuf option
```

### Package Format
```toml
[package]
name = "example"
version = "1.0.0"
description = "Example package"

[dependencies]
libc = ">=1.0"

[build]
type = "make"
script = "build.sh"
```

### Store Path Format
```
/store/packages/<sha256-hash>/
/store/objects/<sha256-prefix>/<sha256-full>
```

---

## Next Steps (Immediate)

1. **Milestone 2.1** - Implement robust IPC dengan reconnection
2. **Milestone 2.2** - Add service health monitoring
3. **Milestone 2.3** - Define package format dan basic repository

---

## Contributing

Lihat `docs/developer-guide.md` untuk panduan kontribusi.

## References

- [OSDev Wiki](https://wiki.osdev.org/)
- [Writing an OS in Rust](https://os.phil-opp.com/)
- [Nix Package Manager](https://nixos.org/)
- [musl libc](https://musl.libc.org/)
