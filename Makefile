# MIXOS GO Build System
# Main Makefile for building the complete operating system

VERSION := 1.0
KERNEL_VERSION := 6.6.10-mixos
ARCH := x86_64

# Directories
ROOT_DIR := $(shell pwd)
BUILD_DIR := $(ROOT_DIR)/build
KERNEL_DIR := $(ROOT_DIR)/kernel
MIX_CLI_DIR := $(ROOT_DIR)/mix-cli
MIX_PKG_DIR := $(ROOT_DIR)/mix-pkg
MIX_AGENT_DIR := $(ROOT_DIR)/mix-agent
MIX_AGENT_EARLY_DIR := $(ROOT_DIR)/mix-agent-early
MIX_INSTALLER_DIR := $(ROOT_DIR)/mix-installer
ROOTFS_DIR := $(ROOT_DIR)/rootfs
INITRAMFS_DIR := $(ROOT_DIR)/initramfs
ISO_DIR := $(ROOT_DIR)/iso

# Output files
KERNEL_OUTPUT := $(BUILD_DIR)/kernel/vmlinuz-$(KERNEL_VERSION)
INITRAMFS_OUTPUT := $(BUILD_DIR)/initramfs/initramfs-$(KERNEL_VERSION).img
ISO_OUTPUT := $(BUILD_DIR)/iso/mixos-go-$(VERSION)-$(ARCH).iso

# Build flags
GO_FLAGS := CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUST_FLAGS := --release

.PHONY: all clean help
.PHONY: kernel mix-cli mix-pkg mix-agent mix-agent-early mix-installer
.PHONY: packages rootfs initramfs iso

# Default target
all: iso
	@echo "=========================================="
	@echo "MIXOS GO Build Complete!"
	@echo "ISO: $(ISO_OUTPUT)"
	@echo "=========================================="

# Help
help:
	@echo "MIXOS GO Build System"
	@echo ""
	@echo "Build Order (correct sequence):"
	@echo "  1. kernel         - Build Linux kernel"
	@echo "  2. mix-cli        - Build CLI tool (Go)"
	@echo "  3. mix-pkg        - Build package manager (Rust)"
	@echo "  4. mix-agent      - Build AI agent (Python)"
	@echo "  5. mix-agent-early- Build early boot agent (Go static)"
	@echo "  6. mix-installer  - Build TUI installer (Go)"
	@echo "  7. packages       - Build all packages"
	@echo "  8. rootfs         - Bootstrap root filesystem"
	@echo "  9. initramfs      - Build initramfs"
	@echo " 10. iso            - Build bootable ISO"
	@echo ""
	@echo "Targets:"
	@echo "  make all          - Build everything"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make <component>  - Build specific component"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  KERNEL_VERSION=$(KERNEL_VERSION)"
	@echo "  ARCH=$(ARCH)"

# Clean
clean:
	@echo "Cleaning build directory..."
	rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Create build directories
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)/{kernel,mix-cli,mix-pkg,mix-agent,mix-agent-early,mix-installer,rootfs,initramfs,iso}

#
# Component builds (in correct order)
#

# 1. Kernel
kernel: $(BUILD_DIR)
	@echo "=========================================="
	@echo "[1/10] Building Kernel"
	@echo "=========================================="
	cd $(KERNEL_DIR) && bash scripts/build-kernel.sh
	@echo "Kernel build complete"

# 2. mix-cli (Go)
mix-cli: $(BUILD_DIR)
	@echo "=========================================="
	@echo "[2/10] Building mix-cli"
	@echo "=========================================="
	cd $(MIX_CLI_DIR) && \
		$(GO_FLAGS) go build -ldflags="-s -w" -o $(BUILD_DIR)/mix-cli/mix-cli .
	@echo "mix-cli build complete"

# 3. mix-pkg (Rust)
mix-pkg: $(BUILD_DIR)
	@echo "=========================================="
	@echo "[3/10] Building mix-pkg"
	@echo "=========================================="
	cd $(MIX_PKG_DIR) && \
		cargo build $(RUST_FLAGS) && \
		cp target/release/mix-pkg $(BUILD_DIR)/mix-pkg/
	@echo "mix-pkg build complete"

# 4. mix-agent (Python)
mix-agent: $(BUILD_DIR)
	@echo "=========================================="
	@echo "[4/10] Building mix-agent"
	@echo "=========================================="
	mkdir -p $(BUILD_DIR)/mix-agent
	cp -r $(MIX_AGENT_DIR)/mixos_agent $(BUILD_DIR)/mix-agent/
	cp $(MIX_AGENT_DIR)/requirements.txt $(BUILD_DIR)/mix-agent/
	cp $(MIX_AGENT_DIR)/pyproject.toml $(BUILD_DIR)/mix-agent/
	@echo "mix-agent build complete"

# 5. mix-agent-early (Go static binary)
mix-agent-early: $(BUILD_DIR)
	@echo "=========================================="
	@echo "[5/10] Building mix-agent-early"
	@echo "=========================================="
	cd $(MIX_AGENT_EARLY_DIR) && \
		$(GO_FLAGS) go build -ldflags="-s -w -extldflags '-static'" \
		-o $(BUILD_DIR)/mix-agent-early/mix-agent-early .
	@echo "mix-agent-early build complete"

# 6. mix-installer (Go)
mix-installer: $(BUILD_DIR)
	@echo "=========================================="
	@echo "[6/10] Building mix-installer"
	@echo "=========================================="
	cd $(MIX_INSTALLER_DIR) && \
		$(GO_FLAGS) go build -ldflags="-s -w" -o $(BUILD_DIR)/mix-installer/mix-installer .
	@echo "mix-installer build complete"

# 7. Packages (all components)
packages: mix-cli mix-pkg mix-agent mix-agent-early mix-installer
	@echo "=========================================="
	@echo "[7/10] All packages built"
	@echo "=========================================="

# 8. RootFS
rootfs: packages
	@echo "=========================================="
	@echo "[8/10] Building RootFS"
	@echo "=========================================="
	cd $(ROOTFS_DIR) && bash bootstrap.sh
	@echo "RootFS build complete"

# 9. Initramfs
initramfs: kernel mix-agent-early
	@echo "=========================================="
	@echo "[9/10] Building Initramfs"
	@echo "=========================================="
	KERNEL_VERSION=$(KERNEL_VERSION) bash $(INITRAMFS_DIR)/scripts/build-initramfs.sh
	@echo "Initramfs build complete"

# 10. ISO
iso: kernel initramfs rootfs
	@echo "=========================================="
	@echo "[10/10] Building ISO"
	@echo "=========================================="
	VERSION=$(VERSION) KERNEL_VERSION=$(KERNEL_VERSION) bash $(ISO_DIR)/scripts/build-iso.sh
	@echo "ISO build complete"

#
# Development targets
#

# Quick rebuild of tools only
tools: mix-cli mix-pkg mix-installer
	@echo "Tools rebuild complete"

# Test builds
test-build:
	@echo "Testing build system..."
	$(MAKE) clean
	$(MAKE) $(BUILD_DIR)
	@echo "Build system test passed"

# Show build status
status:
	@echo "Build Status:"
	@echo "  Kernel:          $(shell [ -f $(KERNEL_OUTPUT) ] && echo 'OK' || echo 'NOT BUILT')"
	@echo "  mix-cli:         $(shell [ -f $(BUILD_DIR)/mix-cli/mix-cli ] && echo 'OK' || echo 'NOT BUILT')"
	@echo "  mix-pkg:         $(shell [ -f $(BUILD_DIR)/mix-pkg/mix-pkg ] && echo 'OK' || echo 'NOT BUILT')"
	@echo "  mix-agent:       $(shell [ -d $(BUILD_DIR)/mix-agent/mixos_agent ] && echo 'OK' || echo 'NOT BUILT')"
	@echo "  mix-agent-early: $(shell [ -f $(BUILD_DIR)/mix-agent-early/mix-agent-early ] && echo 'OK' || echo 'NOT BUILT')"
	@echo "  mix-installer:   $(shell [ -f $(BUILD_DIR)/mix-installer/mix-installer ] && echo 'OK' || echo 'NOT BUILT')"
	@echo "  RootFS:          $(shell [ -d $(BUILD_DIR)/rootfs/work ] && echo 'OK' || echo 'NOT BUILT')"
	@echo "  Initramfs:       $(shell [ -f $(INITRAMFS_OUTPUT) ] && echo 'OK' || echo 'NOT BUILT')"
	@echo "  ISO:             $(shell [ -f $(ISO_OUTPUT) ] && echo 'OK' || echo 'NOT BUILT')"
