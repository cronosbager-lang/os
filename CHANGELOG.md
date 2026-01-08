# Changelog

All notable changes to MIXOS GO will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Complete ISO build infrastructure with BIOS and UEFI support
- ISOLINUX configuration for legacy BIOS boot
- EFI boot structure with GRUB
- GRUB theme for boot menu
- Build automation scripts (build-all.sh, setup-dev-env.sh)
- QEMU test scripts (start-qemu.sh, test-boot.sh, test-install.sh)
- Integration test suite for mix-cli and mix-agent
- Network configuration templates (systemd-networkd)
- Systemd service files for mixos-agent
- First boot configuration script
- Comprehensive documentation (architecture, user-guide, developer-guide, api-reference)
- GitHub Actions CI/CD workflows
- Issue and PR templates

### Changed
- Enhanced build-iso.sh with better error handling and logging
- Improved Makefile with correct build order

### Fixed
- ISO build script now handles missing dependencies gracefully

## [1.0.0] - TBD

### Added
- Initial release of MIXOS GO
- mix-cli: Command-line interface for system management
- mix-pkg: Package manager with multiple backend support (ALPM, dpkg, native)
- mix-agent: AI agent with tool calling, safety validation, and API server
- mix-agent-early: Lightweight AI agent for early boot
- mix-installer: TUI-based installer with autonomous mode
- Custom kernel configuration optimized for containers
- Initramfs with AI-assisted hardware detection
- RootFS with minimal, developer, and full profiles
- Docker integration out of the box

### Components
- **mix-cli** (Go): System management CLI
- **mix-pkg** (Rust): Package manager
- **mix-agent** (Python): AI agent with FastAPI
- **mix-agent-early** (Go): Early boot agent
- **mix-installer** (Go): TUI installer

### AI Features
- Local LLM inference with llama.cpp
- Tool calling for system operations
- Safety validation and sandboxing
- Context management and session tracking
- WebSocket streaming support

### Developer Features
- Pre-configured development environment
- Language runtime setup (Python, Node.js, Go, Rust)
- Project initialization templates
- Container management integration
- VM management support
