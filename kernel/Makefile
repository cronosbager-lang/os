# ============================================================================
# MIXOS GO - Kernel Build Makefile
# ============================================================================
#
# This Makefile is for standalone kernel builds.
# Normally, use the main Makefile in the root directory.
#
# ============================================================================

# Include parent config if available
-include ../config.mk

# Fallback defaults
KERNEL_VERSION ?= 6.6.10
KERNEL_LOCALVERSION ?= -mixos
KERNEL_FULL_VERSION := $(KERNEL_VERSION)$(KERNEL_LOCALVERSION)
KERNEL_CONFIG ?= mixos-default

KERNEL_MAJOR := $(shell echo $(KERNEL_VERSION) | cut -d. -f1)
KERNEL_URL := https://cdn.kernel.org/pub/linux/kernel/v$(KERNEL_MAJOR).x/linux-$(KERNEL_VERSION).tar.xz
KERNEL_DIR := linux-$(KERNEL_VERSION)

ROOT_DIR := $(CURDIR)/..
BUILD_DIR ?= $(ROOT_DIR)/build
KERNEL_BUILD_DIR := $(BUILD_DIR)/kernel
MODULES_DIR := $(KERNEL_BUILD_DIR)/modules
NPROC := $(shell nproc)

# Output files
KERNEL_IMAGE := $(KERNEL_BUILD_DIR)/vmlinuz-$(KERNEL_FULL_VERSION)
KERNEL_MODULES := $(MODULES_DIR)/lib/modules/$(KERNEL_FULL_VERSION)

.PHONY: all download extract configure build modules_install clean distclean help

all: build modules_install

download:
	@echo "==> Downloading Linux kernel $(KERNEL_VERSION)..."
	@if [ ! -f "linux-$(KERNEL_VERSION).tar.xz" ]; then \
		curl -L -o linux-$(KERNEL_VERSION).tar.xz $(KERNEL_URL); \
	fi

extract: download
	@echo "==> Extracting kernel source..."
	@if [ ! -d "$(KERNEL_DIR)" ]; then \
		tar xf linux-$(KERNEL_VERSION).tar.xz; \
	fi

configure: extract
	@echo "==> Configuring kernel with $(KERNEL_CONFIG).config..."
	@mkdir -p $(KERNEL_BUILD_DIR)
	@cp config/$(KERNEL_CONFIG).config $(KERNEL_DIR)/.config
	@echo "$(KERNEL_LOCALVERSION)" > $(KERNEL_DIR)/localversion
	@$(MAKE) -C $(KERNEL_DIR) olddefconfig

menuconfig: extract
	@cp config/$(KERNEL_CONFIG).config $(KERNEL_DIR)/.config 2>/dev/null || true
	@$(MAKE) -C $(KERNEL_DIR) menuconfig
	@cp $(KERNEL_DIR)/.config config/$(KERNEL_CONFIG).config

build: configure
	@echo "==> Building kernel $(KERNEL_FULL_VERSION)..."
	@$(MAKE) -C $(KERNEL_DIR) -j$(NPROC) bzImage
	@$(MAKE) -C $(KERNEL_DIR) -j$(NPROC) modules
	@mkdir -p $(KERNEL_BUILD_DIR)
	@cp $(KERNEL_DIR)/arch/x86/boot/bzImage $(KERNEL_IMAGE)
	@cp $(KERNEL_DIR)/System.map $(KERNEL_BUILD_DIR)/System.map-$(KERNEL_FULL_VERSION)
	@echo "==> Kernel built: $(KERNEL_IMAGE)"

modules_install: build
	@echo "==> Installing modules..."
	@mkdir -p $(MODULES_DIR)
	@$(MAKE) -C $(KERNEL_DIR) INSTALL_MOD_PATH=$(MODULES_DIR) modules_install
	@rm -f $(KERNEL_MODULES)/build $(KERNEL_MODULES)/source
	@echo "==> Modules installed to $(KERNEL_MODULES)"

clean:
	@echo "==> Cleaning..."
	@rm -rf $(KERNEL_DIR)
	@rm -rf $(KERNEL_BUILD_DIR)

distclean: clean
	@rm -f linux-$(KERNEL_VERSION).tar.xz

help:
	@echo "MIXOS Kernel Build"
	@echo ""
	@echo "Targets:"
	@echo "  make all            - Build kernel and install modules (default)"
	@echo "  make download       - Download kernel source"
	@echo "  make extract        - Extract kernel source"
	@echo "  make configure      - Configure kernel"
	@echo "  make build          - Build kernel image"
	@echo "  make modules_install- Install kernel modules"
	@echo "  make menuconfig     - Open kernel config menu"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make distclean      - Clean everything including downloads"
	@echo ""
	@echo "Configuration:"
	@echo "  KERNEL_VERSION      = $(KERNEL_VERSION)"
	@echo "  KERNEL_LOCALVERSION = $(KERNEL_LOCALVERSION)"
	@echo "  KERNEL_FULL_VERSION = $(KERNEL_FULL_VERSION)"
	@echo "  KERNEL_CONFIG       = $(KERNEL_CONFIG)"
	@echo ""
	@echo "Output:"
	@echo "  KERNEL_IMAGE        = $(KERNEL_IMAGE)"
	@echo "  KERNEL_MODULES      = $(KERNEL_MODULES)"
