# ============================================================================
# MIXOS GO - Build Configuration
# Single Source of Truth for all build variables
# ============================================================================

# ----------------------------------------------------------------------------
# VERSION INFORMATION
# ----------------------------------------------------------------------------
VERSION         := 1.0.0
VERSION_MAJOR   := 1
VERSION_MINOR   := 0
VERSION_PATCH   := 0
CODENAME        := alpha
RELEASE_DATE    := $(shell date +%Y%m%d)

# ----------------------------------------------------------------------------
# KERNEL CONFIGURATION
# ----------------------------------------------------------------------------
KERNEL_VERSION       := 6.6.10
KERNEL_LOCALVERSION  := -mixos
KERNEL_FULL_VERSION  := $(KERNEL_VERSION)$(KERNEL_LOCALVERSION)
KERNEL_CONFIG        := mixos-default

# Kernel source URL
KERNEL_MAJOR         := $(shell echo $(KERNEL_VERSION) | cut -d. -f1)
KERNEL_URL           := https://cdn.kernel.org/pub/linux/kernel/v$(KERNEL_MAJOR).x/linux-$(KERNEL_VERSION).tar.xz

# ----------------------------------------------------------------------------
# ARCHITECTURE
# ----------------------------------------------------------------------------
ARCH            := x86_64
ARCH_KERNEL     := x86
HOST_ARCH       := $(shell uname -m)

# ----------------------------------------------------------------------------
# DIRECTORY STRUCTURE
# ----------------------------------------------------------------------------
ROOT_DIR        := $(shell pwd)
BUILD_DIR       := $(ROOT_DIR)/build
CACHE_DIR       := $(BUILD_DIR)/cache
OUTPUT_DIR      := $(BUILD_DIR)/output
WORK_DIR        := $(BUILD_DIR)/work

# Source directories
KERNEL_SRC_DIR      := $(ROOT_DIR)/kernel
INITRAMFS_SRC_DIR   := $(ROOT_DIR)/initramfs
ROOTFS_SRC_DIR      := $(ROOT_DIR)/rootfs
ISO_SRC_DIR         := $(ROOT_DIR)/iso
PACKAGES_SRC_DIR    := $(ROOT_DIR)/packages
CONFIGS_SRC_DIR     := $(ROOT_DIR)/configs
SCRIPTS_SRC_DIR     := $(ROOT_DIR)/scripts

# Component source directories
MIX_CLI_SRC         := $(ROOT_DIR)/mix-cli
MIX_PKG_SRC         := $(ROOT_DIR)/mix-pkg
MIX_AGENT_SRC       := $(ROOT_DIR)/mix-agent
MIX_AGENT_EARLY_SRC := $(ROOT_DIR)/mix-agent-early
MIX_INSTALLER_SRC   := $(ROOT_DIR)/mix-installer

# Build output directories
KERNEL_BUILD_DIR    := $(BUILD_DIR)/kernel
INITRAMFS_BUILD_DIR := $(BUILD_DIR)/initramfs
ROOTFS_BUILD_DIR    := $(BUILD_DIR)/rootfs
ISO_BUILD_DIR       := $(BUILD_DIR)/iso
PACKAGES_BUILD_DIR  := $(BUILD_DIR)/packages

# ----------------------------------------------------------------------------
# OUTPUT FILES (Canonical Paths)
# ----------------------------------------------------------------------------
# Kernel outputs
KERNEL_IMAGE        := $(KERNEL_BUILD_DIR)/vmlinuz-$(KERNEL_FULL_VERSION)
KERNEL_MODULES_DIR  := $(KERNEL_BUILD_DIR)/modules/lib/modules/$(KERNEL_FULL_VERSION)
KERNEL_SYSTEM_MAP   := $(KERNEL_BUILD_DIR)/System.map-$(KERNEL_FULL_VERSION)

# Initramfs output
INITRAMFS_IMAGE     := $(INITRAMFS_BUILD_DIR)/initramfs-$(KERNEL_FULL_VERSION).img
INITRAMFS_WORK      := $(INITRAMFS_BUILD_DIR)/work

# Rootfs outputs
ROOTFS_ROOT         := $(ROOTFS_BUILD_DIR)/root
ROOTFS_SQUASHFS     := $(ROOTFS_BUILD_DIR)/rootfs.sfs

# Package outputs
PKG_MIX_CLI         := $(PACKAGES_BUILD_DIR)/mix-cli/mix-cli
PKG_MIX_PKG         := $(PACKAGES_BUILD_DIR)/mix-pkg/mix-pkg
PKG_MIX_AGENT       := $(PACKAGES_BUILD_DIR)/mix-agent
PKG_MIX_AGENT_EARLY := $(PACKAGES_BUILD_DIR)/mix-agent-early/mix-agent-early
PKG_MIX_INSTALLER   := $(PACKAGES_BUILD_DIR)/mix-installer/mix-installer

# ISO output
ISO_IMAGE           := $(OUTPUT_DIR)/mixos-$(VERSION)-$(ARCH).iso
ISO_WORK            := $(ISO_BUILD_DIR)/work

# ----------------------------------------------------------------------------
# AI MODEL CONFIGURATION
# ----------------------------------------------------------------------------
AI_MODEL_NAME       := mix-small-1.1b
AI_MODEL_QUANT      := q4_k_m
AI_MODEL_FILE       := $(AI_MODEL_NAME)-$(AI_MODEL_QUANT).gguf
AI_MODEL_EARLY_FILE := mix-early-$(AI_MODEL_QUANT).gguf

# AI model paths (these would be downloaded or built separately)
AI_MODEL_PATH       := $(CACHE_DIR)/models/$(AI_MODEL_FILE)
AI_MODEL_EARLY_PATH := $(CACHE_DIR)/models/$(AI_MODEL_EARLY_FILE)

# AI model size limits
AI_MODEL_MAX_SIZE_INITRAMFS := 1500M
AI_MODEL_MAX_SIZE_ROOTFS    := 2000M

# ----------------------------------------------------------------------------
# BUILD FLAGS
# ----------------------------------------------------------------------------
# Compiler flags
CFLAGS          := -O2 -pipe
CXXFLAGS        := -O2 -pipe
LDFLAGS         :=

# Go build flags
GO_VERSION      := 1.25.4
GO_FLAGS        := CGO_ENABLED=0 GOOS=linux GOARCH=amd64
GO_LDFLAGS      := -s -w -X main.Version=$(VERSION)
GO_STATIC_FLAGS := CGO_ENABLED=0 GOOS=linux GOARCH=amd64

# Rust build flags
RUST_FLAGS      := --release
CARGO_FLAGS     := --release

# Kernel build flags
KERNEL_MAKE_FLAGS := -j$(shell nproc)

# ----------------------------------------------------------------------------
# COMPRESSION SETTINGS
# ----------------------------------------------------------------------------
# Initramfs compression
INITRAMFS_COMPRESSION := gzip
INITRAMFS_COMP_LEVEL  := 9

# Squashfs compression
SQUASHFS_COMPRESSION  := zstd
SQUASHFS_COMP_LEVEL   := 19
SQUASHFS_BLOCK_SIZE   := 1M

# ----------------------------------------------------------------------------
# LIVE BOOT CONFIGURATION
# ----------------------------------------------------------------------------
LIVE_LABEL          := MIXOS_LIVE
LIVE_UUID           := $(shell uuidgen 2>/dev/null || echo "MIXOS-LIVE-UUID")
ROOT_LABEL          := MIXOS_ROOT
EFI_LABEL           := MIXOS_EFI

# ----------------------------------------------------------------------------
# ESSENTIAL KERNEL MODULES (for initramfs)
# ----------------------------------------------------------------------------
INITRAMFS_MODULES := \
    ahci \
    sd_mod \
    sr_mod \
    nvme \
    nvme_core \
    cdrom \
    crc16 \
    crc32c \
    ext2 \
    ext3 \
    ext4 \
    jbd2 \
    scsi_mod \
    libata \
    mdiobus \
    mbcache \
    libphy \
    squashfs \
    overlay \
    loop \
    isofs \
    vfat \
    fat \
    nls_cp437 \
    nls_ascii \
    nls_utf8 \
    usb_common \
    usb_core \
    usb_storage \
    uas \
    xhci_hcd \
    xhci_pci \
    ehci_hcd \
    ehci_pci \
    ohci_hcd \
    ohci_pci \
    uhci_hcd \
    virtio \
    virtio_blk \
    virtio_pci \
    virtio_scsi \
    virtio_net \
    e1000 \
    e1000e \
    r8169

# ----------------------------------------------------------------------------
# BASE PACKAGES (for rootfs)
# ----------------------------------------------------------------------------
BASE_PACKAGES := \
    base \
    linux \
    linux-firmware \
    systemd \
    networkmanager \
    docker \
    git \
    vim \
    sudo

# ----------------------------------------------------------------------------
# TOOL PATHS
# ----------------------------------------------------------------------------
BUSYBOX         := $(shell which busybox 2>/dev/null || echo "/bin/busybox")
MKSQUASHFS      := $(shell which mksquashfs 2>/dev/null || echo "mksquashfs")
UNSQUASHFS      := $(shell which unsquashfs 2>/dev/null || echo "unsquashfs")
XORRISO         := $(shell which xorriso 2>/dev/null || echo "xorriso")
GRUB_MKIMAGE    := $(shell which grub-mkimage 2>/dev/null || echo "grub-mkimage")
CPIO            := $(shell which cpio 2>/dev/null || echo "cpio")

# ----------------------------------------------------------------------------
# COLORS FOR OUTPUT
# ----------------------------------------------------------------------------
COLOR_RESET     := \033[0m
COLOR_RED       := \033[0;31m
COLOR_GREEN     := \033[0;32m
COLOR_YELLOW    := \033[1;33m
COLOR_BLUE      := \033[0;34m
COLOR_CYAN      := \033[0;36m
COLOR_WHITE     := \033[1;37m

# ----------------------------------------------------------------------------
# HELPER FUNCTIONS
# ----------------------------------------------------------------------------
define log_info
	@printf "$(COLOR_BLUE)[INFO]$(COLOR_RESET) %s\n" "$(1)"
endef

define log_success
	@printf "$(COLOR_GREEN)[OK]$(COLOR_RESET) %s\n" "$(1)"
endef

define log_warn
	@printf "$(COLOR_YELLOW)[WARN]$(COLOR_RESET) %s\n" "$(1)"
endef

define log_error
	@printf "$(COLOR_RED)[ERROR]$(COLOR_RESET) %s\n" "$(1)"
endef

define log_step
	@printf "$(COLOR_CYAN)[STEP]$(COLOR_RESET) %s\n" "$(1)"
endef

define log_phase
	@printf "\n$(COLOR_WHITE)════════════════════════════════════════════════════════════$(COLOR_RESET)\n"
	@printf "$(COLOR_WHITE) %s$(COLOR_RESET)\n" "$(1)"
	@printf "$(COLOR_WHITE)════════════════════════════════════════════════════════════$(COLOR_RESET)\n\n"
endef

# ----------------------------------------------------------------------------
# VALIDATION
# ----------------------------------------------------------------------------
# Check required tools
REQUIRED_TOOLS := make gcc g++ git curl tar gzip xz cpio
CHECK_TOOLS = $(foreach tool,$(REQUIRED_TOOLS),$(if $(shell which $(tool) 2>/dev/null),,$(error "Required tool '$(tool)' not found")))

# ----------------------------------------------------------------------------
# EXPORT ALL VARIABLES
# ----------------------------------------------------------------------------
export VERSION KERNEL_VERSION KERNEL_FULL_VERSION KERNEL_LOCALVERSION
export ARCH BUILD_DIR ROOT_DIR
export KERNEL_IMAGE KERNEL_MODULES_DIR
export INITRAMFS_IMAGE ROOTFS_ROOT ROOTFS_SQUASHFS ISO_IMAGE
export AI_MODEL_PATH AI_MODEL_EARLY_PATH
export LIVE_LABEL ROOT_LABEL EFI_LABEL
