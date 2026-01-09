# ============================================================================
# MIXOS GO - ISO Build Makefile
# ============================================================================
#
# This Makefile is for standalone ISO builds.
# Normally, use the main Makefile in the root directory.
#
# ============================================================================

# Include parent config if available
-include ../config.mk

# Fallback defaults
VERSION ?= 1.0.0
KERNEL_VERSION ?= 6.6.10
KERNEL_LOCALVERSION ?= -mixos
KERNEL_FULL_VERSION := $(KERNEL_VERSION)$(KERNEL_LOCALVERSION)
ARCH ?= x86_64

ISO_DIR := $(shell pwd)
ROOT_DIR := $(shell dirname $(ISO_DIR))
BUILD_DIR ?= $(ROOT_DIR)/build
ISO_BUILD_DIR := $(BUILD_DIR)/iso
OUTPUT_DIR := $(BUILD_DIR)/output
WORK_DIR := $(ISO_BUILD_DIR)/work

# Input files
KERNEL_IMAGE := $(BUILD_DIR)/kernel/vmlinuz-$(KERNEL_FULL_VERSION)
INITRAMFS_IMAGE := $(BUILD_DIR)/initramfs/initramfs-$(KERNEL_FULL_VERSION).img
ROOTFS_SQUASHFS := $(BUILD_DIR)/rootfs/rootfs.sfs

# Output
ISO_IMAGE := $(OUTPUT_DIR)/mixos-$(VERSION)-$(ARCH).iso

# Labels
LIVE_LABEL := MIXOS_LIVE

.PHONY: all clean iso prepare test test-uefi help

all: iso

clean:
	@echo "Cleaning ISO build..."
	@rm -rf $(ISO_BUILD_DIR)
	@rm -f $(ISO_IMAGE)

prepare:
	@mkdir -p $(WORK_DIR)/{boot,mixos,EFI/BOOT,isolinux}
	@mkdir -p $(OUTPUT_DIR)

iso: prepare
	@echo "Building MIXOS GO ISO..."
	@VERSION=$(VERSION) \
	 KERNEL_FULL_VERSION=$(KERNEL_FULL_VERSION) \
	 KERNEL_IMAGE=$(KERNEL_IMAGE) \
	 INITRAMFS_IMAGE=$(INITRAMFS_IMAGE) \
	 ROOTFS_SQUASHFS=$(ROOTFS_SQUASHFS) \
	 ISO_BUILD_DIR=$(ISO_BUILD_DIR) \
	 OUTPUT_DIR=$(OUTPUT_DIR) \
	 LIVE_LABEL=$(LIVE_LABEL) \
	 bash $(ROOT_DIR)/scripts/build/build-iso.sh

test: $(ISO_IMAGE)
	@echo "Testing ISO with QEMU..."
	@bash $(ROOT_DIR)/scripts/dev/start-qemu.sh -i $(ISO_IMAGE)

test-uefi: $(ISO_IMAGE)
	@echo "Testing ISO with QEMU (UEFI)..."
	@bash $(ROOT_DIR)/scripts/dev/start-qemu.sh -i $(ISO_IMAGE) --uefi

help:
	@echo "MIXOS ISO Build"
	@echo ""
	@echo "Targets:"
	@echo "  make all       - Build ISO (default)"
	@echo "  make iso       - Build ISO"
	@echo "  make prepare   - Prepare work directory"
	@echo "  make test      - Test ISO with QEMU (BIOS)"
	@echo "  make test-uefi - Test ISO with QEMU (UEFI)"
	@echo "  make clean     - Clean build artifacts"
	@echo ""
	@echo "Configuration:"
	@echo "  VERSION           = $(VERSION)"
	@echo "  KERNEL_FULL_VERSION = $(KERNEL_FULL_VERSION)"
	@echo "  ARCH              = $(ARCH)"
	@echo ""
	@echo "Input files:"
	@echo "  KERNEL_IMAGE      = $(KERNEL_IMAGE)"
	@echo "  INITRAMFS_IMAGE   = $(INITRAMFS_IMAGE)"
	@echo "  ROOTFS_SQUASHFS   = $(ROOTFS_SQUASHFS)"
	@echo ""
	@echo "Output:"
	@echo "  ISO_IMAGE         = $(ISO_IMAGE)"
