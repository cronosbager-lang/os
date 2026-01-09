# ============================================================================
# MIXOS GO - Main Build System
# ============================================================================
#
# Build Order (Dependency Chain):
#
#   PHASE 1: kernel
#      │
#      ├──────────────────────────────────┐
#      │                                  │
#      ▼                                  ▼
#   PHASE 2: packages                  (modules)
#      │                                  │
#      ├──────────────┬───────────────────┤
#      │              │                   │
#      ▼              ▼                   ▼
#   PHASE 3:     initramfs            rootfs
#                    │                   │
#                    │                   ▼
#                    │              rootfs-squash
#                    │                   │
#                    └─────────┬─────────┘
#                              │
#                              ▼
#                    PHASE 4: iso
#
# ============================================================================

# Include configuration
include config.mk

# ============================================================================
# PHONY TARGETS
# ============================================================================
.PHONY: all clean distclean help
.PHONY: kernel kernel-config kernel-menuconfig
.PHONY: packages mix-cli mix-pkg mix-agent mix-agent-early mix-installer
.PHONY: initramfs rootfs rootfs-squash
.PHONY: iso
.PHONY: check-deps check-tools validate
.PHONY: status info
.PHONY: test test-qemu test-boot

# ============================================================================
# DEFAULT TARGET
# ============================================================================
all: iso
	$(call log_phase,MIXOS GO Build Complete!)
	$(call log_success,ISO Image: $(ISO_IMAGE))
	@echo ""
	@echo "To test with QEMU:"
	@echo "  make test-qemu"
	@echo ""

# ============================================================================
# HELP
# ============================================================================
help:
	@echo ""
	@echo "╔══════════════════════════════════════════════════════════════════╗"
	@echo "║                    MIXOS GO Build System                         ║"
	@echo "╚══════════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "Build Targets (in dependency order):"
	@echo "  make kernel          - Build Linux kernel and modules"
	@echo "  make packages        - Build all MIXOS packages"
	@echo "  make initramfs       - Build initramfs with AI components"
	@echo "  make rootfs          - Build root filesystem"
	@echo "  make rootfs-squash   - Compress rootfs to squashfs"
	@echo "  make iso             - Build bootable ISO image"
	@echo "  make all             - Build everything (default)"
	@echo ""
	@echo "Individual Package Targets:"
	@echo "  make mix-cli         - Build CLI tool (Go)"
	@echo "  make mix-pkg         - Build package manager (Rust)"
	@echo "  make mix-agent       - Prepare AI agent (Python)"
	@echo "  make mix-agent-early - Build early boot agent (Go static)"
	@echo "  make mix-installer   - Build TUI installer (Go)"
	@echo ""
	@echo "Utility Targets:"
	@echo "  make clean           - Remove build artifacts"
	@echo "  make distclean       - Remove all generated files"
	@echo "  make status          - Show build status"
	@echo "  make info            - Show configuration info"
	@echo "  make check-deps      - Check build dependencies"
	@echo "  make test-qemu       - Test ISO in QEMU"
	@echo ""
	@echo "Configuration:"
	@echo "  VERSION         = $(VERSION)"
	@echo "  KERNEL_VERSION  = $(KERNEL_FULL_VERSION)"
	@echo "  ARCH            = $(ARCH)"
	@echo "  BUILD_DIR       = $(BUILD_DIR)"
	@echo ""

# ============================================================================
# DIRECTORY CREATION
# ============================================================================
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

$(KERNEL_BUILD_DIR): | $(BUILD_DIR)
	@mkdir -p $(KERNEL_BUILD_DIR)

$(PACKAGES_BUILD_DIR): | $(BUILD_DIR)
	@mkdir -p $(PACKAGES_BUILD_DIR)/{mix-cli,mix-pkg,mix-agent,mix-agent-early,mix-installer}

$(INITRAMFS_BUILD_DIR): | $(BUILD_DIR)
	@mkdir -p $(INITRAMFS_BUILD_DIR)

$(ROOTFS_BUILD_DIR): | $(BUILD_DIR)
	@mkdir -p $(ROOTFS_BUILD_DIR)

$(ISO_BUILD_DIR): | $(BUILD_DIR)
	@mkdir -p $(ISO_BUILD_DIR)

$(OUTPUT_DIR): | $(BUILD_DIR)
	@mkdir -p $(OUTPUT_DIR)

$(CACHE_DIR): | $(BUILD_DIR)
	@mkdir -p $(CACHE_DIR)/{downloads,models}

# ============================================================================
# PHASE 1: KERNEL BUILD
# ============================================================================
.PHONY: kernel-download kernel-extract kernel-configure kernel-build kernel-modules

kernel: $(KERNEL_IMAGE) $(KERNEL_MODULES_DIR)
	$(call log_success,Kernel build complete)

$(KERNEL_IMAGE) $(KERNEL_MODULES_DIR): | $(KERNEL_BUILD_DIR) $(CACHE_DIR)
	$(call log_phase,PHASE 1: Building Kernel $(KERNEL_FULL_VERSION))
	@bash $(SCRIPTS_SRC_DIR)/build/build-kernel.sh

kernel-menuconfig: | $(KERNEL_BUILD_DIR)
	$(call log_info,Opening kernel menuconfig...)
	@cd $(KERNEL_SRC_DIR) && $(MAKE) menuconfig

# ============================================================================
# PHASE 2: PACKAGES BUILD
# ============================================================================
packages: mix-cli mix-pkg mix-agent mix-agent-early mix-installer
	$(call log_phase,PHASE 2: All Packages Built)

# --- mix-cli (Go) ---
mix-cli: $(PKG_MIX_CLI)
$(PKG_MIX_CLI): $(MIX_CLI_SRC)/main.go $(MIX_CLI_SRC)/go.mod | $(PACKAGES_BUILD_DIR)
	$(call log_step,Building mix-cli...)
	@mkdir -p $(dir $@)
	@cd $(MIX_CLI_SRC) && \
		$(GO_STATIC_FLAGS) go build \
		-ldflags="$(GO_LDFLAGS)" \
		-o $@ .
	$(call log_success,mix-cli built: $@)

# --- mix-pkg (Rust) ---
mix-pkg: $(PKG_MIX_PKG)
$(PKG_MIX_PKG): $(MIX_PKG_SRC)/Cargo.toml $(wildcard $(MIX_PKG_SRC)/src/*.rs) | $(PACKAGES_BUILD_DIR)
	$(call log_step,Building mix-pkg...)
	@mkdir -p $(dir $@)
	@cd $(MIX_PKG_SRC) && cargo build $(CARGO_FLAGS)
	@cp $(MIX_PKG_SRC)/target/release/mix-pkg $@
	$(call log_success,mix-pkg built: $@)

# --- mix-agent (Python - copy) ---
mix-agent: $(PKG_MIX_AGENT)/mixos_agent/__init__.py
$(PKG_MIX_AGENT)/mixos_agent/__init__.py: $(MIX_AGENT_SRC)/mixos_agent/__init__.py | $(PACKAGES_BUILD_DIR)
	$(call log_step,Preparing mix-agent...)
	@mkdir -p $(PKG_MIX_AGENT)
	@cp -r $(MIX_AGENT_SRC)/mixos_agent $(PKG_MIX_AGENT)/
	@cp $(MIX_AGENT_SRC)/requirements.txt $(PKG_MIX_AGENT)/
	@cp $(MIX_AGENT_SRC)/pyproject.toml $(PKG_MIX_AGENT)/
	$(call log_success,mix-agent prepared: $(PKG_MIX_AGENT))

# --- mix-agent-early (Go static) ---
mix-agent-early: $(PKG_MIX_AGENT_EARLY)
$(PKG_MIX_AGENT_EARLY): $(MIX_AGENT_EARLY_SRC)/main.go $(MIX_AGENT_EARLY_SRC)/go.mod | $(PACKAGES_BUILD_DIR)
	$(call log_step,Building mix-agent-early (static)...)
	@mkdir -p $(dir $@)
	@cd $(MIX_AGENT_EARLY_SRC) && \
		$(GO_STATIC_FLAGS) go build \
		-ldflags="$(GO_LDFLAGS) -extldflags '-static'" \
		-o $@ .
	$(call log_success,mix-agent-early built: $@)

# --- mix-installer (Go) ---
mix-installer: $(PKG_MIX_INSTALLER)
$(PKG_MIX_INSTALLER): $(MIX_INSTALLER_SRC)/main.go $(MIX_INSTALLER_SRC)/go.mod | $(PACKAGES_BUILD_DIR)
	$(call log_step,Building mix-installer...)
	@mkdir -p $(dir $@)
	@cd $(MIX_INSTALLER_SRC) && \
		$(GO_STATIC_FLAGS) go build \
		-ldflags="$(GO_LDFLAGS)" \
		-o $@ .
	$(call log_success,mix-installer built: $@)

# ============================================================================
# PHASE 3A: INITRAMFS BUILD
# ============================================================================
initramfs: $(INITRAMFS_IMAGE)
$(INITRAMFS_IMAGE): $(KERNEL_IMAGE) $(KERNEL_MODULES_DIR) $(PKG_MIX_AGENT_EARLY) | $(INITRAMFS_BUILD_DIR)
	$(call log_phase,PHASE 3A: Building Initramfs)
	@bash $(SCRIPTS_SRC_DIR)/build/build-initramfs.sh
	$(call log_success,Initramfs built: $@)

# ============================================================================
# PHASE 3B: ROOTFS BUILD
# ============================================================================
rootfs: $(ROOTFS_ROOT)/etc/os-release
$(ROOTFS_ROOT)/etc/os-release: $(KERNEL_MODULES_DIR) packages | $(ROOTFS_BUILD_DIR)
	$(call log_phase,PHASE 3B: Building Root Filesystem)
	@bash $(SCRIPTS_SRC_DIR)/build/build-rootfs.sh
	$(call log_success,Rootfs built: $(ROOTFS_ROOT))

# ============================================================================
# PHASE 3C: ROOTFS COMPRESSION
# ============================================================================
rootfs-squash: $(ROOTFS_SQUASHFS)
$(ROOTFS_SQUASHFS): $(ROOTFS_ROOT)/etc/os-release
	$(call log_step,Compressing rootfs to squashfs...)
	@bash $(SCRIPTS_SRC_DIR)/build/build-squashfs.sh
	$(call log_success,Squashfs created: $@)

# ============================================================================
# PHASE 4: ISO BUILD
# ============================================================================
iso: $(ISO_IMAGE)
$(ISO_IMAGE): $(KERNEL_IMAGE) $(INITRAMFS_IMAGE) $(ROOTFS_SQUASHFS) | $(ISO_BUILD_DIR) $(OUTPUT_DIR)
	$(call log_phase,PHASE 4: Building ISO Image)
	@bash $(SCRIPTS_SRC_DIR)/build/build-iso.sh
	$(call log_success,ISO built: $@)

# ============================================================================
# CLEAN TARGETS
# ============================================================================
clean:
	$(call log_info,Cleaning build directory...)
	@rm -rf $(BUILD_DIR)
	$(call log_success,Clean complete)

distclean: clean
	$(call log_info,Cleaning all generated files...)
	@rm -rf $(KERNEL_SRC_DIR)/linux-$(KERNEL_VERSION)
	@rm -f $(KERNEL_SRC_DIR)/linux-$(KERNEL_VERSION).tar.xz
	@cd $(MIX_PKG_SRC) && cargo clean 2>/dev/null || true
	$(call log_success,Distclean complete)

# ============================================================================
# UTILITY TARGETS
# ============================================================================
check-deps:
	$(call log_info,Checking build dependencies...)
	@bash $(SCRIPTS_SRC_DIR)/build/check-deps.sh

check-tools:
	$(call log_info,Checking required tools...)
	@for tool in $(REQUIRED_TOOLS); do \
		if ! command -v $$tool >/dev/null 2>&1; then \
			echo "Missing: $$tool"; \
			exit 1; \
		fi; \
	done
	$(call log_success,All required tools found)

status:
	@echo ""
	@echo "╔══════════════════════════════════════════════════════════════════╗"
	@echo "║                      Build Status                                ║"
	@echo "╚══════════════════════════════════════════════════════════════════╝"
	@echo ""
	@printf "  %-20s %s\n" "Kernel:" "$$([ -f $(KERNEL_IMAGE) ] && echo '✓ $(KERNEL_IMAGE)' || echo '✗ Not built')"
	@printf "  %-20s %s\n" "Kernel Modules:" "$$([ -d $(KERNEL_MODULES_DIR) ] && echo '✓ $(KERNEL_MODULES_DIR)' || echo '✗ Not built')"
	@printf "  %-20s %s\n" "mix-cli:" "$$([ -f $(PKG_MIX_CLI) ] && echo '✓ Built' || echo '✗ Not built')"
	@printf "  %-20s %s\n" "mix-pkg:" "$$([ -f $(PKG_MIX_PKG) ] && echo '✓ Built' || echo '✗ Not built')"
	@printf "  %-20s %s\n" "mix-agent:" "$$([ -d $(PKG_MIX_AGENT)/mixos_agent ] && echo '✓ Prepared' || echo '✗ Not prepared')"
	@printf "  %-20s %s\n" "mix-agent-early:" "$$([ -f $(PKG_MIX_AGENT_EARLY) ] && echo '✓ Built' || echo '✗ Not built')"
	@printf "  %-20s %s\n" "mix-installer:" "$$([ -f $(PKG_MIX_INSTALLER) ] && echo '✓ Built' || echo '✗ Not built')"
	@printf "  %-20s %s\n" "Initramfs:" "$$([ -f $(INITRAMFS_IMAGE) ] && echo '✓ $(INITRAMFS_IMAGE)' || echo '✗ Not built')"
	@printf "  %-20s %s\n" "Rootfs:" "$$([ -f $(ROOTFS_ROOT)/etc/os-release ] && echo '✓ $(ROOTFS_ROOT)' || echo '✗ Not built')"
	@printf "  %-20s %s\n" "Squashfs:" "$$([ -f $(ROOTFS_SQUASHFS) ] && echo '✓ $(ROOTFS_SQUASHFS)' || echo '✗ Not built')"
	@printf "  %-20s %s\n" "ISO:" "$$([ -f $(ISO_IMAGE) ] && echo '✓ $(ISO_IMAGE)' || echo '✗ Not built')"
	@echo ""

info:
	@echo ""
	@echo "╔══════════════════════════════════════════════════════════════════╗"
	@echo "║                    Build Configuration                           ║"
	@echo "╚══════════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "Version Information:"
	@printf "  %-25s %s\n" "MIXOS Version:" "$(VERSION)"
	@printf "  %-25s %s\n" "Codename:" "$(CODENAME)"
	@printf "  %-25s %s\n" "Kernel Version:" "$(KERNEL_FULL_VERSION)"
	@printf "  %-25s %s\n" "Architecture:" "$(ARCH)"
	@echo ""
	@echo "Paths:"
	@printf "  %-25s %s\n" "Root Directory:" "$(ROOT_DIR)"
	@printf "  %-25s %s\n" "Build Directory:" "$(BUILD_DIR)"
	@printf "  %-25s %s\n" "Output Directory:" "$(OUTPUT_DIR)"
	@echo ""
	@echo "Output Files:"
	@printf "  %-25s %s\n" "Kernel Image:" "$(KERNEL_IMAGE)"
	@printf "  %-25s %s\n" "Initramfs:" "$(INITRAMFS_IMAGE)"
	@printf "  %-25s %s\n" "Rootfs Squashfs:" "$(ROOTFS_SQUASHFS)"
	@printf "  %-25s %s\n" "ISO Image:" "$(ISO_IMAGE)"
	@echo ""

# ============================================================================
# TEST TARGETS
# ============================================================================
test-qemu: $(ISO_IMAGE)
	$(call log_info,Starting QEMU test...)
	@bash $(SCRIPTS_SRC_DIR)/dev/start-qemu.sh $(ISO_IMAGE)

test-qemu-uefi: $(ISO_IMAGE)
	$(call log_info,Starting QEMU test (UEFI)...)
	@bash $(SCRIPTS_SRC_DIR)/dev/start-qemu.sh $(ISO_IMAGE) --uefi

test-boot: $(ISO_IMAGE)
	$(call log_info,Running boot test...)
	@bash $(ROOT_DIR)/tests/qemu/test-boot.sh $(ISO_IMAGE)

# ============================================================================
# DEVELOPMENT TARGETS
# ============================================================================
.PHONY: dev-setup dev-shell

dev-setup:
	$(call log_info,Setting up development environment...)
	@bash $(SCRIPTS_SRC_DIR)/dev/setup-dev-env.sh

dev-shell:
	$(call log_info,Starting development shell...)
	@bash --rcfile $(SCRIPTS_SRC_DIR)/dev/dev-bashrc
